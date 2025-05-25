package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
)

type PrimeRequest struct {
	// Method must always contain "isPrime".
	Method *string `json:"method"`
	// Number is any valid JSON number, including floating-point values.
	Number *float64 `json:"number"`
}

type PrimeResponse struct {
	// Method must always contain "isPrime".
	Method string `json:"method"`
	// Prime is true if the number in the PrimeRequest was prime, false if it was not.
	Prime bool `json:"prime"`
}

// TODO: Polish.
func PrimeTime(conn net.Conn) error {
	defer CloseOrLog(conn)

	scanner := bufio.NewScanner(conn)
	// After connecting, a client may send multiple requests in a single session.
	// Each request should be handled in order.
	for scanner.Scan() {
		// Each request is a single line containing a JSON object, terminated by a newline character ('\n', or ASCII 10).
		line := scanner.Text()
		var r PrimeRequest
		err := json.Unmarshal([]byte(line), &r)

		// A request is malformed if:
		// - it is not a well-formed JSON object
		// - if any required field is missing
		// - if the method name is not "isPrime"
		// - or if the number value is not a number.

		switch {
		case err != nil:
			resp, _ := json.Marshal(PrimeResponse{Method: "malformed"})
			conn.Write(append(resp, '\n'))
			return nil
		case r.Method == nil || r.Number == nil:
			resp, _ := json.Marshal(PrimeResponse{Method: "malformed"})
			conn.Write(append(resp, '\n'))
			return nil
		case *r.Method != "isPrime":
			resp, _ := json.Marshal(PrimeResponse{Method: "malformed"})
			conn.Write(append(resp, '\n'))
			return nil
		default:
			resp, _ := json.Marshal(PrimeResponse{Method: "isPrime", Prime: isPrime(*r.Number)})
			conn.Write(append(resp, '\n'))
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	return nil
}

func isPrime(number float64) bool {
	return big.NewInt(int64(number)).ProbablyPrime(0)
}
