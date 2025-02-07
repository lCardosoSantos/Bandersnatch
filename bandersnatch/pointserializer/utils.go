package pointserializer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

const FIELDNAME_PARTIAL_READ = bandersnatchErrors.FIELDNAME_PARTIAL_READ
const FIELDNAME_PARTIAL_WRITE = bandersnatchErrors.FIELDNAME_PARTIAL_WRITE
const FIELDNAME_ACTUALLY_READ = bandersnatchErrors.FIELDNAME_ACTUALLY_READ
const FIELDNAME_BYTES_READ = bandersnatchErrors.FIELDNAME_BYTES_READ

// additional data contained in errors returned by consumeExpectRead. Note that this "extends" bandersnatchErrors.ReadErrorData
type headerRead struct {
	PartialRead    bool
	ActuallyRead   []byte
	ExpectedToRead []byte
	BytesRead      int
}

func init() {
	errorsWithData.CheckIsSubtype[bandersnatchErrors.ReadErrorData, headerRead]() // ensure that headerRead extends bandersnatchErrors.ReadErorData
}

const ErrorPrefix = "bandersnatch / serialization: "

var ErrDidNotReadExpectedString = bandersnatchErrors.ErrDidNotReadExpectedString

// Our code below makes use of formatting in the form %v{FieldName}. If we ever refactor field names, this would break.
// This init - routine panics if we change field names to alert to this.
func init() {
	errorsWithData.CheckParameterForStruct[bandersnatchErrors.ReadErrorData]("PartialRead")
	errorsWithData.CheckParameterForStruct[bandersnatchErrors.ReadErrorData]("BytesRead")
	errorsWithData.CheckParameterForStruct[bandersnatchErrors.ReadErrorData]("ActuallyRead")
	errorsWithData.CheckParameterForStruct[bandersnatchErrors.WriteErrorData]("BytesWritten")
	errorsWithData.CheckParameterForStruct[bandersnatchErrors.WriteErrorData]("PartialWrite")
	errorsWithData.CheckParameterForStruct[headerRead]("ExpectedToRead")
	errorsWithData.CheckParameterForStruct[headerRead]("PartialRead")
	errorsWithData.CheckParameterForStruct[headerRead]("ActuallyRead")
	errorsWithData.CheckParameterForStruct[headerRead]("BytesRead")
}

// consumeExpectRead reads and consumes len(expectToRead) bytes from input and reports an error if the read bytes differ from expectToRead.
// This is intended to read headers. Remember to use errors.Is to check the returned errors rather than == due to error wrapping.
//
// NOTES:
// Returns an error wrapping io.ErrUnexpectedEOF or io.EOF on end-of-file (io.EOF if the io.Reader was in EOF state to start with, io.ErrUnexpectedEOF if we encounter EOF after reading >0 bytes)
// On mismatch of expectToRead vs. actually read values, returns an error wrapping ErrDidNotReadExpectedString
//
// Panics if expectToRead has length >MaxInt32. The function always (tries to) consume len(expectToRead) bytes, even if a mismatch is already early in the stream.
// Panics if expectToRead is nil or input is nil (unless len(expectToRead)==0)
//
// The returned error type satisfies the error interface and, if non-nil, contains an instance of headerRead,
// ActuallyRead contains the actually read bytes (type []byte)
// PartialRead (type bool) is true iff 0 < bytes_read < len(expectToRead).
// Note here that if bytesRead == len(expectToRead), io errors are dropped and the only possible error is ErrDidNotReadExpectedString.
//
// Possible errors (modulo wrapping):
// io errors, io.EOF, io.ErrUnexpectedEOF, ErrDidNotReadExpectedString
func consumeExpectRead(input io.Reader, expectToRead []byte) (bytes_read int, returnedError errorsWithData.ErrorWithGuaranteedParameters[headerRead]) {
	// We do not treat nil as an empyt byte slice here. This is an internal function and we expect ourselves to behave properly: nil indicates a bug.
	if expectToRead == nil {
		panic(ErrorPrefix + "consumeExpectRead called with nil input for expectToRead")
	}
	l := len(expectToRead) // number of bytes we will try to read
	// Ensure l fits into an int32 (signed!)
	if l > math.MaxInt32 {
		// should we return an error instead of panicking?
		panic(fmt.Errorf(ErrorPrefix+"trying to read from io.Reader, expecting to read %v bytes, which is more than MaxInt32", l))
	}

	// consuming 0 bytes is always successful, even if the input io.Reader is actually invalid (e.g. nil)
	if l == 0 {
		return 0, nil
	}

	if input == nil {
		panic(ErrorPrefix + "consumeExpectRead was called on nil reader")
	}

	var err error
	var buf []byte = make([]byte, l)
	bytes_read, err = io.ReadFull(input, buf) // read l bytes into buffer

	if err != nil {
		buf = buf[0:bytes_read:bytes_read] // We reduce the cap, so maybe some future version of Go actually frees the trailing memory. (We *could* copy it to a new buffer, but that's probably worse in most cases)

		// Note: We deep-copy the contents of expectToRead. The reason is that the caller might later modify the backing array otherwise.
		var returnedErrorData headerRead = headerRead{ActuallyRead: buf, ExpectedToRead: copyByteSlice(expectToRead), BytesRead: bytes_read} // extra data returned in error

		if errors.Is(err, io.ErrUnexpectedEOF) {
			// Note: Sprintf is only used for the length of the expected input. The other %%v{arg} are done via errorsWithData, hence escaping the %.
			message := fmt.Sprintf(ErrorPrefix+"Unexpected EOF after reading %%v{BytesRead} out of %v bytes when reading header.\nReported error was %%w.\nBytes expected were 0x%%x{ExpectedToRead}, got 0x%%x{ActuallyRead}", len(expectToRead))
			returnedErrorData.PartialRead = true
			returnedError = errorsWithData.NewErrorWithParametersFromData(err, message, &returnedErrorData)
			return
		} else if errors.Is(err, io.EOF) {
			message := ErrorPrefix + "EOF when trying to read buffer.\nExpected to read 0x%x{ExpectedToRead}, got EOF instead"

			// Note: bytes_read == 0 is guaranteed by io.ReadFull if the error is io.EOF

			returnedErrorData.ActuallyRead = make([]byte, 0)
			returnedErrorData.PartialRead = false

			returnedError = errorsWithData.NewErrorWithParametersFromData(err, message, &returnedErrorData)
			return
		} else { // other io error
			returnedErrorData.PartialRead = (bytes_read > 0) // NOTE: io.ReadFull guarantees bytes_read < l, since errors after full reads are dropped.
			returnedError = errorsWithData.NewErrorWithParametersFromData(err, "", &returnedErrorData)
			return
		}

	}

	// We successfully read len(expectToRead) many bytes. Now check if they match.
	if !bytes.Equal(expectToRead, buf) {
		// Note: We deep-copy the contents of expectToRead. The reason is that the caller might later modify the backing array otherwise, which screws up the error message.
		var returnedErrorData headerRead = headerRead{ActuallyRead: buf, ExpectedToRead: copyByteSlice(expectToRead), BytesRead: bytes_read, PartialRead: false} // extra data returned in error

		err = bandersnatchErrors.ErrDidNotReadExpectedString
		message := ErrorPrefix + "Unexpected Header encountered upon deserialization. Expected 0x%x{ExpectedToRead}, got 0x%x{ActuallyRead}"
		returnedError = errorsWithData.NewErrorWithParametersFromData(err, message, &returnedErrorData)
		return
	}
	returnedError = nil // this is true anyway at this point; just added for clarity.
	return
}

// Note: This returns a copy (by design). For v==nil, we return a fresh, empty non-nil slice.

// copyByteSlice returns a copy of the given byte slice (with newly allocated underlying array).
// For nil inputs, returns an empty byte slice.
func copyByteSlice(v []byte) (ret []byte) {
	if v == nil {
		ret = make([]byte, 0)
		return
	}
	ret = make([]byte, len(v))
	copy(ret, v)
	return
}

// writeFull(output, data) wraps around output.Write(data) by adding error data.
//
// On error, the returned error has an extra data field in addition to WriteErrorData (accessible via errorsWithData) called "Data" that holds (a deep copy of) the data that we tried to write.
func writeFull(output io.Writer, data []byte) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	// Note: output.Write may interpret a nil byte slice as an empty []byte array and actually work.
	// However, since this is an internal function and we never intend to call it with something that may be nil, we panic.
	if data == nil {
		panic(ErrorPrefix + "called writeFull with nil byte slice")
	}

	bytesWritten, errPlain := output.Write(data)
	if errPlain != nil {
		err = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.WriteErrorData](errPlain, ErrorPrefix+"An error occured when trying to write %v{Data} to io.Writer. We only wrote %v{BytesWritten} data. The error was:\n%w",
			"Data", copyByteSlice(data),
			bandersnatchErrors.FIELDNAME_BYTES_WRITTEN, bytesWritten,
			FIELDNAME_PARTIAL_WRITE, bytesWritten != 0 && bytesWritten < len(data),
		)
	} // else err == nil
	return
}
