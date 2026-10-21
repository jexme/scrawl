#ifndef SCRAWL_CAPTURE_H
#define SCRAWL_CAPTURE_H

#include <stdint.h>

// Result codes of sec_capture.
enum {
	SEC_OK = 0,
	SEC_TIMEOUT = 1,  // the frame did not arrive within the allotted time
	SEC_STREAM = 2,   // stream error, text in err
	SEC_SETUP = 3,    // could not connect to PipeWire, text in err
};

void sec_init(void);

// sec_capture connects to node through the already open fd (fd passes into the
// ownership of the PipeWire library and is closed inside), waits for the first frame and
// returns it as RGBA in *pixels (malloc memory, freed by the caller).
int sec_capture(int fd, uint32_t node, int timeout_ms, uint8_t **pixels,
                int *width, int *height, char *err, int errlen);

#endif
