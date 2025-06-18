package audit

import (
	"path"
	"runtime"
	"strconv"
)

func Originator() string {
	pc, source, line, ok := runtime.Caller(1)
	if !ok {
		return ""
	}

	signature := path.Base(runtime.FuncForPC(pc).Name())

	return signature + "." + source + "#L " + strconv.Itoa(line)
}
