package core

// #cgo CFLAGS: -I ./lib
// #cgo LDFLAGS: -L ./lib -l clash -Wl,-rpath -Wl,./lib
// #include "libclash.h"
// #include <stdlib.h>
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"
)

func NodeTest(mapping map[string]interface{}, url string, msTimeout int) (delay int, err error) {
	delay, _, err = nodeTest(mapping, url, msTimeout, false)
	return delay, err
}

func NodeTestWithResponse(mapping map[string]interface{}, url string, msTimeout int) (delay int, data []byte, err error) {
	return nodeTest(mapping, url, msTimeout, true)
}

func nodeTest(mapping map[string]interface{}, url string, timeout int, readBody bool) (int, []byte, error) {
	size := int(unsafe.Sizeof(C.struct_request{}))
	request := (*C.struct_request)(C.malloc(C.size_t(size)))
	request.url = C.CString(url)
	request.method = C.CString("GET")
	defer C.free(unsafe.Pointer(request))

	raw, _ := json.Marshal(mapping)
	nodeInfo := C.CString(string(raw))
	delay := (*C.int)(unsafe.Pointer(new(C.int)))
	response := (**C.char)(unsafe.Pointer(nil))
	responseLen := (*C.int)(unsafe.Pointer(nil))
	errMessage := (**C.char)(unsafe.Pointer(new(*C.char)))
	errLen := (*C.int)(unsafe.Pointer(new(C.int)))

	if readBody {
		response = (**C.char)(unsafe.Pointer(new(*C.char)))
		responseLen = (*C.int)(unsafe.Pointer(new(C.int)))
	}

	C.HttpTest(nodeInfo, request, C.int(timeout), delay, response, responseLen, errMessage, errLen)
	if *errMessage != nil {
		defer C.free(unsafe.Pointer(*errMessage))
		return 0, nil, fmt.Errorf("%s", C.GoBytes(unsafe.Pointer(*errMessage), *errLen))
	}

	if readBody && *response != nil {
		defer C.free(unsafe.Pointer(*response))
		return int(*delay), C.GoBytes(unsafe.Pointer(*response), *responseLen), nil
	}

	return int(*delay), nil, nil
}
