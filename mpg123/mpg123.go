// mpg123.go contains all bindings to the C library

package mpg123

/*
#define MPG123_ENUM_API 1
#include <mpg123.h>
#cgo CFLAGS: -I/usr/local/include
#cgo LDFLAGS: -L/usr/local/lib -lmpg123

int do_mpg123_read(mpg123_handle *mh, void *outmemory, size_t outmemsize, size_t *done) {
	return mpg123_read(mh, outmemory, outmemsize, done);
}
*/
import "C"

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"unsafe"
)

var EOF = errors.New("EOF")

// All output encoding formats supported by mpg123
const (
	ENC_8           = C.MPG123_ENC_8
	ENC_16          = C.MPG123_ENC_16
	ENC_24          = C.MPG123_ENC_24
	ENC_32          = C.MPG123_ENC_32
	ENC_SIGNED      = C.MPG123_ENC_SIGNED
	ENC_FLOAT       = C.MPG123_ENC_FLOAT
	ENC_SIGNED_8    = C.MPG123_ENC_SIGNED_8
	ENC_UNSIGNED_8  = C.MPG123_ENC_UNSIGNED_8
	ENC_ULAW_8      = C.MPG123_ENC_ULAW_8
	ENC_ALAW_8      = C.MPG123_ENC_ALAW_8
	ENC_SIGNED_16   = C.MPG123_ENC_SIGNED_16
	ENC_UNSIGNED_16 = C.MPG123_ENC_UNSIGNED_16
	ENC_SIGNED_24   = C.MPG123_ENC_SIGNED_24
	ENC_UNSIGNED_24 = C.MPG123_ENC_UNSIGNED_24
	ENC_SIGNED_32   = C.MPG123_ENC_SIGNED_32
	ENC_UNSIGNED_32 = C.MPG123_ENC_UNSIGNED_32
	ENC_FLOAT_32    = C.MPG123_ENC_FLOAT_32
	ENC_FLOAT_64    = C.MPG123_ENC_FLOAT_64
	ENC_ANY         = C.MPG123_ENC_ANY

	ADD_FLAGS = C.MPG123_ADD_FLAGS
	QUIET     = C.MPG123_QUIET
)

const (
	IN_MAX_BUFFER_SIZE  = 16384
	OUT_MAX_BUFFER_SIZE = 32768
)

// Contains a handle for and mpg123 decoder instance
type Decoder struct {
	handle *C.mpg123_handle
	io.Seeker
}

// init initializes the mpg123 library when package is loaded
func init() {
	err := C.mpg123_init()
	if err != C.MPG123_OK {
		//return fmt.Errorf("error initializing mpg123")
		panic("failed to initialize mpg123")
	}
	//return nil
}

///////////////////////////
// DECODER INITIAL CODE //
///////////////////////////

func InitializeMpg123() {
	C.mpg123_init()
	return
}

func ExitMpg123() {
	C.mpg123_exit()
	return
}

///////////////////////////
// DECODER INSTANCE CODE //
///////////////////////////

// NewDecoder creates a new mpg123 decoder instance
func NewDecoder(decoder string, params ...C.long) (*Decoder, error) {
	var err C.int
	var mh *C.mpg123_handle
	if decoder == "" {
		mh = C.mpg123_new(nil, &err)
	} else {
		cdecoder := C.CString(decoder)
		defer C.free(unsafe.Pointer(cdecoder))
		mh = C.mpg123_new(cdecoder, &err)
		if params != nil {
			C.mpg123_param(mh, C.MPG123_FLAGS, params[0], 0.)
		}
	}
	if mh == nil {
		errstring := C.mpg123_plain_strerror(err)
		err := C.GoString(errstring)
		C.free(unsafe.Pointer(errstring))
		return nil, fmt.Errorf("error initializing mpg123 decoder: %s", err)
	}
	dec := new(Decoder)
	dec.handle = mh
	return dec, nil
}

// Delete frees an mpg123 decoder instance
func (d *Decoder) Delete() {
	C.mpg123_delete(d.handle)
}

// returns a string containing the most recent error message corresponding to
// an mpg123 decoder instance
func (d *Decoder) strerror() string {
	return C.GoString(C.mpg123_strerror(d.handle))
}

////////////////////////
// OUTPUT FORMAT CODE //
////////////////////////

// FormatNone disables all decoder output formats (used to specifying supported formats)
func (d *Decoder) FormatNone() {
	C.mpg123_format_none(d.handle)
}

// FromatAll enables all decoder output formats (this is the default setting)
func (d *Decoder) FormatAll() {
	C.mpg123_format_all(d.handle)
}

// GetFormat returns current output format
func (d *Decoder) GetFormat() (rate int, channels int, encoding int) {
	var cRate C.long
	var cChans, cEnc C.int
	C.mpg123_getformat(d.handle, &cRate, &cChans, &cEnc)
	return int(cRate), int(cChans), int(cEnc)
}

// Format sets the audio output format for decoder
func (d *Decoder) Format(rate int, channels int, encodings int) {
	C.mpg123_format(d.handle, C.long(rate), C.int(channels), C.int(encodings))
}

/////////////////////////////
// INPUT AND DECODING CODE //
/////////////////////////////

// Open initializes a decoder for an mp3 file using a filename
func (d *Decoder) Open(file string) error {
	cfile := C.CString(file)
	defer C.free(unsafe.Pointer(cfile))
	err := C.mpg123_open(d.handle, cfile)
	if err != C.MPG123_OK {
		return fmt.Errorf("error opening %s: %s", file, d.strerror())
	}
	return nil
}

// OpenFile binds to an fd from an open *os.File for decoding
func (d *Decoder) OpenFile(f *os.File) error {
	err := C.mpg123_open_fd(d.handle, C.int(f.Fd()))
	if err != C.MPG123_OK {
		return fmt.Errorf("error attaching file: %s", d.strerror())
	}
	return nil
}

// OpenFeed prepares a decoder for direct feeding via Feed(..)
func (d *Decoder) OpenFeed() error {
	err := C.mpg123_open_feed(d.handle)
	if err != C.MPG123_OK {
		return fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	return nil
}

// Close closes an input file if one was opened by mpg123
func (d *Decoder) Close() error {
	err := C.mpg123_close(d.handle)
	if err != C.MPG123_OK {
		return fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	return nil
}

// Read decodes data and into buf and returns number of bytes decoded.
func (d *Decoder) Read(buf []byte) (int, error) {
	var done C.size_t
	err := C.do_mpg123_read(d.handle, (unsafe.Pointer)(&buf[0]), C.size_t(len(buf)), &done)
	if err == C.MPG123_DONE {
		return int(done), EOF
	}
	if err != C.MPG123_OK {
		return int(done), fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	return int(done), nil
}

func (d *Decoder) ReadAudioFrames(frames int, buf []byte) (int, error) {
	var done C.size_t
	_, channels, enc := d.GetFormat()
	bytesPerSample := GetEncodingBitsPerSample(enc) / 8
	framesToBytes := bytesPerSample * frames * channels
	err := C.do_mpg123_read(d.handle, (unsafe.Pointer)(&buf[0]), C.size_t(framesToBytes), &done)
	if err == C.MPG123_DONE {
		return int(done), EOF
	}
	if err != C.MPG123_OK {
		return int(done), fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	return int(done), nil
}

func (d *Decoder) DecodeSamples(samples int, audio []byte) (int, error) {
	rLen, err := d.ReadAudioFrames(samples, audio)
	if err == EOF {
		return 0, nil
	}
	return (rLen / 4), nil
}

// Feed provides data bytes into the decoder
func (d *Decoder) Feed(buf []byte) error {
	err := C.mpg123_feed(d.handle, (*C.uchar)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
	if err != C.MPG123_OK {
		return fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	return nil
}

// DecoderReader is the way to decode streaming MP3
type DecoderReader struct {
	decoder  *Decoder
	src      io.Reader
	fps      int
	channels int
	paranoid bool
}

// Paranoid mode shuts off the decoder on a non-EOF error (handy if your input is a duplex network stream).
func (dr *DecoderReader) Paranoid() *DecoderReader {
	dr.paranoid = true
	return dr
}

// Nuke kills our DecoderReader appropriately
func (dr DecoderReader) Nuke() {
	dr.decoder.Close()
	// dr.decoder.Delete() // Commented-out because it causes a SIGABRT 😰
}

// Read duck-types DecoderReader into io.Reader.
func (dr DecoderReader) Read(bytes []byte) (int, error) {
	buf := make([]byte, 64*1024)
	for {
		var n int
		var err error

		// Feed data
		if n, err = dr.src.Read(buf); err == nil {
			if err = dr.decoder.Feed(buf[0:n]); err != nil {
				log.Print("Error while feeding to mpg123: ", err)
			}
		} else if dr.paranoid {
			// Note: EOF in Feed does NOT mean EOF in Read!
			dr.Nuke()
			return 0, err
		}

		// Read output
		var done C.size_t
		msg := C.mpg123_read(dr.decoder.handle, (*C.uchar)(&bytes[0]), C.size_t(len(bytes)), &done)
		switch msg {
		case C.MPG123_NEW_FORMAT:
			rate, channel, encoding := dr.decoder.GetFormat()
			log.Printf(
				"New format with rate: %d, channels: %d, encoding: %d", rate, channel, encoding,
			)
			fallthrough
		case C.MPG123_OK:
			fallthrough
		case C.MPG123_DONE:
			fallthrough
		case C.MPG123_NEED_MORE:
			if done > 0 {
				return int(done), nil
			}
			if err == io.EOF {
				// Source exhausted, so signal EOF
				dr.Nuke()
				return int(done), io.EOF
			}
		}
	}
}

// DecoderReader gives you an io.Reader for streaming-decoding. It performs
// a combination of Feed and Read, and relies on you to first call OpenFeed
// before invoking DecoderReader.Read.
func (d *Decoder) DecoderReader(
	src io.Reader, fps int, channels int, encoding int,
) *DecoderReader {
	d.FormatNone()
	d.Format(int64(fps), channels, encoding)
	return &DecoderReader{
		decoder:  d,
		src:      src,
		fps:      fps,
		channels: channels,
		paranoid: false,
	}
}

// MonoDecoderReader is an alias that gives you an io.Reader for
// decoding a stream that is known to be mono-channeled.
func (d *Decoder) MonoDecoderReader(src io.Reader, fps int, encoding int) *DecoderReader {
	return d.DecoderReader(src, fps, 1, encoding)
}

// Feed input chunk and get first chunk of decoded audio.
func (d *Decoder) Decode(buf []byte) ([]byte, error) {
	var b bytes.Buffer
	out := make([]byte, OUT_MAX_BUFFER_SIZE)
	var outLen int
	var size C.size_t

	ret := C.mpg123_decode(d.handle, (*C.uchar)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), (*C.uchar)(&out[0]), C.size_t(OUT_MAX_BUFFER_SIZE), &size)
	if ret == C.MPG123_NEW_FORMAT {
		var rate C.long
		var channels, enc C.int

		C.mpg123_getformat(d.handle, &rate, &channels, &enc)
		log.Printf("New format: %d Hz, %d channels, encoding value %d\n", rate, channels, enc)
	} else if ret == C.MPG123_ERR || ret == C.MPG123_NEED_MORE {
		log.Printf("mpg123 first decode error!!!\n")
		return nil, fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	outLen = int(size)
	if outLen > 0 {
		b.Write(out[:outLen])
		log.Printf("mpg123 first decode. %d\n", outLen)
	}

	for {
		ret = C.mpg123_decode(d.handle, nil, 0, (*C.uchar)(&out[0]), C.size_t(OUT_MAX_BUFFER_SIZE), &size)
		if ret == C.MPG123_ERR || ret == C.MPG123_NEED_MORE {
			break
		}
		outLen = int(size)
		if outLen > 0 {
			b.Write(out[:outLen])
		}
	}

	if ret == C.MPG123_ERR {
		log.Printf("mpg123 decode error!!!\n")
		return nil, fmt.Errorf("mpg123 error: %s", d.strerror())
	}

	return b.Bytes(), nil
}

// const char* mpg123_current_decoder(mpg123_handle *mh)
func (d *Decoder) CurrentDecoder() string {
	dec := C.mpg123_current_decoder(d.handle)
	return C.GoString(dec)
}

func (d *Decoder) Seek(offset int64, whence int) (int64, error) {
	c_offset := (C.off_t)(offset)
	c_whence := (C.int)(whence)
	s_offset := (int64)(C.mpg123_seek(d.handle, c_offset, c_whence))
	return s_offset, nil
}

// const char** mpg123_supported_decoders(void)
func SupportedDecoders() []string {
	dec := C.mpg123_supported_decoders()

	var strings []string
	q := uintptr(unsafe.Pointer(dec))
	for {
		dec = (**C.char)(unsafe.Pointer(q))
		if *dec == nil {
			break
		}
		strings = append(strings, C.GoString(*dec))
		q += unsafe.Sizeof(q)
	}

	return strings
}

// off_t mpg123_tell(mpg123_handle *mh)
func (d *Decoder) TellCurrentSample() int64 {
	return int64(C.mpg123_tell(d.handle))
}

// int mpg123_encsize	(	int 	encoding	)
func GetEncodingBitsPerSample(encoding int) int {
	return 8 * int(C.mpg123_encsize(C.int(encoding)))
}

// off_t mpg123_length(mpg123_handle * 	mh)
func (d *Decoder) GetLengthInPCMFrames() int {
	return int(C.mpg123_length(d.handle))
}

// Param sets a specific parameter on an mpg123 handle.
func (d *Decoder) Param(paramType int, value int64, fvalue float64) error {
	err := C.mpg123_param(d.handle, uint32(paramType), C.long(value), C.double(fvalue))
	if err != C.MPG123_OK {
		return fmt.Errorf("mpg123 error: %s", d.strerror())
	}
	return nil
}
