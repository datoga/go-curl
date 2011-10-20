package curl

/*
#include <stdlib.h>
#include <curl/curl.h>
#include "callback.h"
static CURLcode curl_easy_setopt_long(CURL *handle, CURLoption option, long parameter) {
  return curl_easy_setopt(handle, option, parameter);
}
static CURLcode curl_easy_setopt_string(CURL *handle, CURLoption option, char *parameter) {
  return curl_easy_setopt(handle, option, parameter);
}
static CURLcode curl_easy_setopt_slist(CURL *handle, CURLoption option, struct curl_slist *parameter) {
  return curl_easy_setopt(handle, option, parameter);
}
static CURLcode curl_easy_setopt_pointer(CURL *handle, CURLoption option, void *parameter) {
  return curl_easy_setopt(handle, option, parameter);
}

// get info
static CURLcode curl_easy_getinfo_string(CURL *curl, CURLINFO info, char **p) {
 return curl_easy_getinfo(curl, info, p);
}
static CURLcode curl_easy_getinfo_long(CURL *curl, CURLINFO info, long *p) {
 return curl_easy_getinfo(curl, info, p);
}
static CURLcode curl_easy_getinfo_double(CURL *curl, CURLINFO info, double *p) {
 return curl_easy_getinfo(curl, info, p);
}
static CURLcode curl_easy_getinfo_slist(CURL *curl, CURLINFO info, struct curl_slist *p) {
 return curl_easy_getinfo(curl, info, p);
}

*/
import "C"

import (
	"unsafe"
	"reflect"
	"os"
	"fmt"
)

/*
*/
// consts
/*
const (
	OPTTYPE_LONG          = 0
	OPTTYPE_OBJECTPOINT   = 10000
	OPTTYPE_FUNCTIONPOINT = 20000
	OPTTYPE_OFF_T         = 30000
)
*/

const (
	PAUSE_RECV      = C.CURLPAUSE_RECV
	PAUSE_RECV_CONT = C.CURLPAUSE_RECV_CONT
	PAUSE_SEND      = C.CURLPAUSE_SEND
	PAUSE_SEND_CONT = C.CURLPAUSE_SEND_CONT
	PAUSE_ALL       = C.CURLPAUSE_ALL
	PAUSE_CONT      = C.CURLPAUSE_CONT
)

// ======================== functions ========



// all ret code
type CurlError C.CURLcode

func (e CurlError) String() string {
	// ret is const char*, no need to free
	ret := C.curl_easy_strerror(C.CURLcode(e))
	return fmt.Sprintf("curl: %s", C.GoString(ret))
}

func newCurlError(errno C.CURLcode) os.Error {
	if errno == C.CURLE_OK {		// if nothing wrong
		return nil
	}
	return CurlError(errno)
}


// int (*curl_progress_callback)(void *clientp, double dltotal, double dlnow, double ultotal, double ulnow);
// size_t writefunction_static_func( char *ptr, size_t size, size_t nmemb, void *userdata);

// curl_easy interface
type CURL struct {
	handle unsafe.Pointer
	onDataAvailable, onHeaderAvailable func([]byte, uintptr, interface{}) uintptr
	onProgressAvailable func(interface{}, float64, float64, float64, float64) int
}

func EasyInit() *CURL {
	p := C.curl_easy_init()
	return &CURL{p, nil, nil, nil}
}

func (curl *CURL) Duphandle() *CURL {
	p := curl.handle
	return &CURL{C.curl_easy_duphandle(p), nil, nil, nil}
}

func (curl *CURL) Cleanup() {
	p := curl.handle
	C.curl_easy_cleanup(p)
}

func (curl *CURL) Perform() os.Error {
	p := curl.handle
	return newCurlError(C.curl_easy_perform(p))
}

// export this function to c
//export callWriteFunctionCallback
func callWriteFunctionCallback(
	f func([]byte, uintptr, interface{}) uintptr,
	ptr *C.char,
	size C.size_t,
	userdata interface{}) uintptr {
	// TODO: avoid C char -> Go sting -> go []Byte
	buf := []byte(C.GoStringN(ptr, C.int(size)))
	ret := f(buf, uintptr(size), userdata)
	return ret
}

// WARNING: why ? function pointer is &fun, but function addr is reflect.ValueOf(fun).Pointer()
func (curl *CURL) Setopt(opt int, param interface{}) os.Error {
	p := curl.handle
	switch {
	case opt == OPT_READFUNCTION:
		panic("readfunction not implemented yet!")
	case opt == OPT_HEADERFUNCTION:
		fun := param.(func([]byte, uintptr, interface{}) uintptr)
		curl.onHeaderAvailable = fun

		ptr := C.return_sample_callback(unsafe.Pointer(reflect.ValueOf(fun).Pointer()))
		if err := newCurlError(C.curl_easy_setopt_pointer(p, C.CURLoption(opt), ptr)); err == nil {
			return newCurlError(C.curl_easy_setopt_pointer(p, OPT_HEADERDATA, unsafe.Pointer(reflect.ValueOf(fun).Pointer())))
		} else {
			return err
		}

	case opt == OPT_WRITEFUNCTION:
		fun := param.(func([]byte, uintptr, interface{}) uintptr)
		curl.onDataAvailable = fun

		ptr := C.return_sample_callback(unsafe.Pointer(reflect.ValueOf(fun).Pointer()))
		return newCurlError(C.curl_easy_setopt_pointer(p, C.CURLoption(opt), ptr))

	case opt > C.CURLOPTTYPE_OFF_T:
		// here we should use uint64
		panic("> off_t not implemented yet!")
	case opt > C.CURLOPTTYPE_FUNCTIONPOINT:
		// function pointer
		panic("function poionter not implemented yet!")
		break
	case opt > C.CURLOPTTYPE_OBJECTPOINT:
		switch t := param.(type) {
		case string:
			// FIXME: memory leak, some opt needs we hold a c string till perform()
			ptr := C.CString(t)
			// defer C.free(unsafe.Pointer(ptr))
			ret := C.curl_easy_setopt_string(p, C.CURLoption(opt), ptr)
			return newCurlError(ret)
		case []string:
			println("DEBUG(setopt)", "creating a list")
			if len(t) > 0 {
				a_slist := C.curl_slist_append(nil, C.CString(t[0]))
				for _, s := range t[1:] {
					a_slist = C.curl_slist_append(a_slist, C.CString(s))
				}
				return newCurlError(C.curl_easy_setopt_slist(p, C.CURLoption(opt), a_slist))
			} else {
				return newCurlError(C.curl_easy_setopt_slist(p, C.CURLoption(opt), nil))
			}
		default:
			// It panics if v's Kind is not Chan, Func, Map, Ptr, Slice, or UnsafePointer.
			// val := reflect.ValueOf(param)
			//fmt.Printf("DEBUG(Setopt): param=%x\n", val.Pointer())
			//println("DEBUG can addr =", val.Pointer(), "opt=", opt)
			// pass a pointer to GoInterface
			return newCurlError(C.curl_easy_setopt_pointer(p, C.CURLoption(opt),
				unsafe.Pointer(&param)))
		}
	case opt > C.CURLOPTTYPE_LONG:
		// long
		switch t := param.(type) {
		case int:
			val := C.long(t)
			ret := C.curl_easy_setopt_long(p, C.CURLoption(opt), val)
			return newCurlError(ret)
		case bool:
			val := 0
			if t {
				val = 1
			}
			ret := C.curl_easy_setopt_long(p, C.CURLoption(opt), C.long(val))
			return newCurlError(ret)
		default:
			panic("type error in param")
		}
	}
	panic("opt param error!")
	return newCurlError(1)
}

// TODO: curl_easy_recv
// TODO: curl_easy_send


func (curl *CURL) Pause(bitmask int) os.Error {
	p := curl.handle
	return newCurlError(C.curl_easy_pause(p, C.int(bitmask)))
}

func (curl *CURL) Reset() {
	p := curl.handle
	C.curl_easy_reset(p)
}

func (curl *CURL) Escape(url string) string {
	p := curl.handle
	oldUrl := C.CString(url)
	defer C.free(unsafe.Pointer(oldUrl))
	newUrl := C.curl_easy_escape(p, oldUrl, 0)
	defer C.curl_free(unsafe.Pointer(newUrl))
	return C.GoString(newUrl)
}

func (curl *CURL) Unescape(url string) string {
	p := curl.handle
	oldUrl := C.CString(url)
	outlength := C.int(0)
	defer C.free(unsafe.Pointer(oldUrl))
	// If outlength is non-NULL, the function will write the length of the
	// returned string in  the  integer  it  points  to.  This allows an
	// escaped string containing %00 to still get used properly after unescaping.
	newUrl := C.curl_easy_unescape(p, oldUrl, 0, &outlength)
	defer C.curl_free(unsafe.Pointer(newUrl))
	return C.GoStringN(newUrl, outlength)
}

/*
 CURLINFO_STRING   0x100000
 CURLINFO_LONG     0x200000
 CURLINFO_DOUBLE   0x300000
 CURLINFO_SLIST    0x400000
 CURLINFO_MASK     0x0fffff
 CURLINFO_TYPEMASK 0xf00000
 */

const (
	_INFO_STRING = C.CURLINFO_STRING
	_INFO_LONG = C.CURLINFO_LONG
	_INFO_DOUBLE = C.CURLINFO_DOUBLE
	_INFO_SLIST = C.CURLINFO_SLIST
	_INFO_MASK = C.CURLINFO_MASK
	_INFO_TYPEMASK = C.CURLINFO_TYPEMASK
)

func (curl *CURL) Getinfo(info C.CURLINFO) (ret interface{}, err os.Error) {
	p := curl.handle
	switch info & _INFO_TYPEMASK {
	case _INFO_STRING:
		a_string := C.CString("")
		defer C.free(unsafe.Pointer(a_string))
		err := newCurlError(C.curl_easy_getinfo_string(p, info, &a_string));
		ret := C.GoString(a_string)
		print("debug (Getinfo) ", ret, "\n")
		return ret, err
	case _INFO_LONG:
		a_long := C.long(-1)
		err := newCurlError(C.curl_easy_getinfo_long(p, info, &a_long));
		ret := int(a_long)
		print("debug (Getinfo) ", ret, "\n")
		return ret, err
	case _INFO_DOUBLE:
		a_double := C.double(0.0)
		err := newCurlError(C.curl_easy_getinfo_double(p, info, &a_double));
		ret := float64(a_double)
		print("debug (Getinfo) ", ret, "\n")
		return ret, err
	case _INFO_SLIST:			// need fix
		a_ptr_slist := new(_Ctype_struct_curl_slist)
		err := newCurlError(C.curl_easy_getinfo_slist(p, info, a_ptr_slist));
		ret := []string{}
		for a_ptr_slist != nil {
			print("!!debug (Getinfo) ", C.GoString(a_ptr_slist.data), a_ptr_slist.next, "\n")
			ret = append(ret, C.GoString(a_ptr_slist.data))
			a_ptr_slist = a_ptr_slist.next
		}
		return ret, err
	default:
		panic("error calling Getinfo\n")
	}
	panic("not implemented yet!")
	return nil, nil
}
