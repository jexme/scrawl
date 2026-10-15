#include "capture.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

#include <pipewire/pipewire.h>
#include <spa/param/video/format-utils.h>
#include <spa/param/video/raw.h>
#include <spa/param/buffers.h>
#include <spa/pod/builder.h>

#define PENDING (-1)

struct capture {
	struct pw_main_loop *loop;
	struct pw_stream *stream;
	struct spa_source *timer;

	struct spa_video_info_raw info;
	int have_format;

	int result; // PENDING, пока кадр не получен и ошибок не было
	uint8_t *pixels;
	int width, height;
	char *err;
	int errlen;
};

static void finish(struct capture *c, int code, const char *msg)
{
	if (c->result != PENDING)
		return;
	c->result = code;
	if (msg && c->err && c->errlen > 0)
		snprintf(c->err, c->errlen, "%s", msg);
	pw_main_loop_quit(c->loop);
}

static void on_state_changed(void *data, enum pw_stream_state old,
                             enum pw_stream_state state, const char *error)
{
	struct capture *c = data;
	(void)old;
	if (state == PW_STREAM_STATE_ERROR)
		finish(c, SEC_STREAM, error ? error : "ошибка потока PipeWire");
}

// После согласования формата просим обычную память (memfd/memptr): dma-buf без
// драйвера GPU не прочитать.
static void on_param_changed(void *data, uint32_t id, const struct spa_pod *param)
{
	struct capture *c = data;
	uint32_t media_type, media_subtype;
	uint8_t buf[512];
	struct spa_pod_builder b = SPA_POD_BUILDER_INIT(buf, sizeof(buf));
	const struct spa_pod *params[1];
	int stride;

	if (param == NULL || id != SPA_PARAM_Format)
		return;
	if (spa_format_parse(param, &media_type, &media_subtype) < 0)
		return;
	if (media_type != SPA_MEDIA_TYPE_video || media_subtype != SPA_MEDIA_SUBTYPE_raw)
		return;
	if (spa_format_video_raw_parse(param, &c->info) < 0)
		return;
	c->have_format = 1;

	stride = SPA_ROUND_UP_N((int)c->info.size.width * 4, 4);
	params[0] = spa_pod_builder_add_object(&b,
		SPA_TYPE_OBJECT_ParamBuffers, SPA_PARAM_Buffers,
		SPA_PARAM_BUFFERS_buffers, SPA_POD_CHOICE_RANGE_Int(4, 2, 8),
		SPA_PARAM_BUFFERS_blocks, SPA_POD_Int(1),
		SPA_PARAM_BUFFERS_size, SPA_POD_Int(stride * (int)c->info.size.height),
		SPA_PARAM_BUFFERS_stride, SPA_POD_Int(stride),
		SPA_PARAM_BUFFERS_dataType, SPA_POD_CHOICE_FLAGS_Int(
			(1 << SPA_DATA_MemFd) | (1 << SPA_DATA_MemPtr)));
	pw_stream_update_params(c->stream, params, 1);
}

// Копирует кадр в RGBA (BGRx/BGRA -> RGBA, альфа непрозрачная).
static void on_process(void *data)
{
	struct capture *c = data;
	struct pw_buffer *pb;
	struct spa_data *d;
	uint32_t w, h, y, x;
	int stride;
	const uint8_t *src;
	uint8_t *dst;

	if (c->result != PENDING)
		return;
	pb = pw_stream_dequeue_buffer(c->stream);
	if (!pb)
		return;

	d = &pb->buffer->datas[0];
	if (c->have_format && d->data != NULL && d->chunk->size > 0 &&
	    !(d->chunk->flags & SPA_CHUNK_FLAG_CORRUPTED)) {
		w = c->info.size.width;
		h = c->info.size.height;
		stride = d->chunk->stride > 0 ? d->chunk->stride : (int)w * 4;
		src = (const uint8_t *)d->data + d->chunk->offset;

		// Буфер должен вместить все строки, иначе чтение выйдет за его край.
		if ((size_t)stride * (h - 1) + (size_t)w * 4 <= d->maxsize - d->chunk->offset) {
			c->pixels = malloc((size_t)w * h * 4);
			if (c->pixels) {
				for (y = 0; y < h; y++) {
					const uint8_t *s = src + (size_t)y * stride;
					dst = c->pixels + (size_t)y * w * 4;
					for (x = 0; x < w; x++, s += 4, dst += 4) {
						dst[0] = s[2];
						dst[1] = s[1];
						dst[2] = s[0];
						dst[3] = 255;
					}
				}
				c->width = (int)w;
				c->height = (int)h;
				finish(c, SEC_OK, NULL);
			}
		}
	}
	pw_stream_queue_buffer(c->stream, pb);
}

static void on_timeout(void *data, uint64_t expirations)
{
	(void)expirations;
	finish((struct capture *)data, SEC_TIMEOUT, "кадр не пришёл вовремя");
}

static const struct pw_stream_events stream_events = {
	PW_VERSION_STREAM_EVENTS,
	.state_changed = on_state_changed,
	.param_changed = on_param_changed,
	.process = on_process,
};

void sec_init(void)
{
	pw_init(NULL, NULL);
}

int sec_capture(int fd, uint32_t node, int timeout_ms, uint8_t **pixels,
                int *width, int *height, char *err, int errlen)
{
	struct capture c;
	struct pw_context *context = NULL;
	struct pw_core *core = NULL;
	struct spa_hook listener;
	uint8_t buf[1024];
	struct spa_pod_builder b = SPA_POD_BUILDER_INIT(buf, sizeof(buf));
	const struct spa_pod *params[1];
	struct timespec ts;
	int result;

	memset(&c, 0, sizeof(c));
	c.result = PENDING;
	c.err = err;
	c.errlen = errlen;
	if (err && errlen > 0)
		err[0] = '\0';

	c.loop = pw_main_loop_new(NULL);
	if (!c.loop) {
		close(fd);
		snprintf(err, errlen, "не удалось создать цикл PipeWire");
		return SEC_SETUP;
	}
	context = pw_context_new(pw_main_loop_get_loop(c.loop), NULL, 0);
	if (!context) {
		close(fd);
		pw_main_loop_destroy(c.loop);
		snprintf(err, errlen, "не удалось создать контекст PipeWire");
		return SEC_SETUP;
	}
	core = pw_context_connect_fd(context, fd, NULL, 0); // забирает fd себе
	if (!core) {
		pw_context_destroy(context);
		pw_main_loop_destroy(c.loop);
		snprintf(err, errlen, "не удалось подключиться к PipeWire по fd");
		return SEC_SETUP;
	}

	c.stream = pw_stream_new(core, "screen-editor",
		pw_properties_new(PW_KEY_MEDIA_TYPE, "Video",
		                  PW_KEY_MEDIA_CATEGORY, "Capture",
		                  PW_KEY_MEDIA_ROLE, "Screen", NULL));
	if (!c.stream) {
		pw_core_disconnect(core);
		pw_context_destroy(context);
		pw_main_loop_destroy(c.loop);
		snprintf(err, errlen, "не удалось создать поток PipeWire");
		return SEC_SETUP;
	}
	pw_stream_add_listener(c.stream, &listener, &stream_events, &c);

	params[0] = spa_pod_builder_add_object(&b,
		SPA_TYPE_OBJECT_Format, SPA_PARAM_EnumFormat,
		SPA_FORMAT_mediaType, SPA_POD_Id(SPA_MEDIA_TYPE_video),
		SPA_FORMAT_mediaSubtype, SPA_POD_Id(SPA_MEDIA_SUBTYPE_raw),
		SPA_FORMAT_VIDEO_format, SPA_POD_CHOICE_ENUM_Id(3,
			SPA_VIDEO_FORMAT_BGRx, SPA_VIDEO_FORMAT_BGRx, SPA_VIDEO_FORMAT_BGRA),
		SPA_FORMAT_VIDEO_size, SPA_POD_CHOICE_RANGE_Rectangle(
			&SPA_RECTANGLE(320, 240), &SPA_RECTANGLE(1, 1), &SPA_RECTANGLE(16384, 16384)),
		SPA_FORMAT_VIDEO_framerate, SPA_POD_CHOICE_RANGE_Fraction(
			&SPA_FRACTION(25, 1), &SPA_FRACTION(0, 1), &SPA_FRACTION(1000, 1)));

	if (pw_stream_connect(c.stream, PW_DIRECTION_INPUT, node,
	                      PW_STREAM_FLAG_AUTOCONNECT | PW_STREAM_FLAG_MAP_BUFFERS,
	                      params, 1) < 0) {
		finish(&c, SEC_STREAM, "pw_stream_connect не удался");
	} else {
		c.timer = pw_loop_add_timer(pw_main_loop_get_loop(c.loop), on_timeout, &c);
		ts.tv_sec = timeout_ms / 1000;
		ts.tv_nsec = (long)(timeout_ms % 1000) * 1000000L;
		pw_loop_update_timer(pw_main_loop_get_loop(c.loop), c.timer, &ts, NULL, false);
		pw_main_loop_run(c.loop);
	}

	if (c.timer)
		pw_loop_destroy_source(pw_main_loop_get_loop(c.loop), c.timer);
	spa_hook_remove(&listener);
	pw_stream_destroy(c.stream);
	pw_core_disconnect(core);
	pw_context_destroy(context);
	pw_main_loop_destroy(c.loop);

	result = c.result == PENDING ? SEC_STREAM : c.result;
	if (result == SEC_OK) {
		*pixels = c.pixels;
		*width = c.width;
		*height = c.height;
	} else {
		free(c.pixels);
	}
	return result;
}
