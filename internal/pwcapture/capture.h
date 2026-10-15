#ifndef SCREEN_EDITOR_CAPTURE_H
#define SCREEN_EDITOR_CAPTURE_H

#include <stdint.h>

// Коды результата sec_capture.
enum {
	SEC_OK = 0,
	SEC_TIMEOUT = 1,  // кадр не пришёл за отведённое время
	SEC_STREAM = 2,   // ошибка потока, текст в err
	SEC_SETUP = 3,    // не удалось подключиться к PipeWire, текст в err
};

void sec_init(void);

// sec_capture подключается к узлу node через уже открытый fd (fd переходит во
// владение библиотеки PipeWire и закрывается внутри), ждёт первый кадр и
// отдаёт его как RGBA в *pixels (память malloc, освобождает вызывающий).
int sec_capture(int fd, uint32_t node, int timeout_ms, uint8_t **pixels,
                int *width, int *height, char *err, int errlen);

#endif
