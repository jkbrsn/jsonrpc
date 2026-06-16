package jsonrpc

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithAllowJSONRPC1_Response(t *testing.T) {
	// Bitcoin Core-style JSON-RPC 1.0 response: no "jsonrpc" field, result and a null error.
	const btc = `{"result":953931,"error":null,"id":1}`

	t.Run("1.0 response decodes with option", func(t *testing.T) {
		resp, err := DecodeResponse([]byte(btc), WithAllowJSONRPC1())
		require.NoError(t, err)
		assert.Equal(t, "", resp.Version(), "decoded version preserved as-is")
		assert.Nil(t, resp.Err())

		var n int64
		require.NoError(t, resp.UnmarshalResult(&n))
		assert.Equal(t, int64(953931), n)
	})

	t.Run("1.0 response rejected without option", func(t *testing.T) {
		_, err := DecodeResponse([]byte(btc))
		require.Error(t, err)
	})

	t.Run("explicit 1.0 version string decodes with option", func(t *testing.T) {
		resp, err := DecodeResponse([]byte(`{"jsonrpc":"1.0","result":42,"error":null,"id":1}`), WithAllowJSONRPC1())
		require.NoError(t, err)
		assert.Equal(t, "1.0", resp.Version())
	})

	t.Run("error:null with result decodes for 2.0 unconditionally", func(t *testing.T) {
		resp, err := DecodeResponse([]byte(`{"jsonrpc":"2.0","result":7,"error":null,"id":1}`))
		require.NoError(t, err)
		assert.Nil(t, resp.Err())
		var n int64
		require.NoError(t, resp.UnmarshalResult(&n))
		assert.Equal(t, int64(7), n)
	})

	t.Run("real error still decodes as error", func(t *testing.T) {
		resp, err := DecodeResponse([]byte(`{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}`))
		require.NoError(t, err)
		require.NotNil(t, resp.Err())
		assert.Equal(t, -32601, resp.Err().Code)
	})

	t.Run("1.0 error response with null result decodes as error", func(t *testing.T) {
		resp, err := DecodeResponse([]byte(`{"result":null,"error":{"code":-1,"message":"boom"},"id":1}`), WithAllowJSONRPC1())
		require.NoError(t, err)
		require.NotNil(t, resp.Err())
		assert.Equal(t, -1, resp.Err().Code)
	})

	t.Run("2.0 null result without error stays a valid success", func(t *testing.T) {
		resp, err := DecodeResponse([]byte(`{"jsonrpc":"2.0","result":null,"id":1}`))
		require.NoError(t, err)
		assert.Nil(t, resp.Err())
		assert.Equal(t, "null", string(resp.RawResult()))
	})

	t.Run("encode of 1.0-decoded response stays strict 2.0", func(t *testing.T) {
		resp, err := DecodeResponse([]byte(btc), WithAllowJSONRPC1())
		require.NoError(t, err)
		_, err = resp.MarshalJSON()
		require.Error(t, err, "marshaling a non-2.0 response must fail")
	})
}

func TestWithAllowJSONRPC1_Request(t *testing.T) {
	// JSON-RPC 1.0 request: no "jsonrpc" field, positional params, integer id.
	const req10 = `{"method":"getblockcount","params":[],"id":1}`

	t.Run("1.0 request decodes with option", func(t *testing.T) {
		req, err := DecodeRequest([]byte(req10), WithAllowJSONRPC1())
		require.NoError(t, err)
		assert.Equal(t, "", req.JSONRPC)
		assert.Equal(t, "getblockcount", req.Method)
		assert.Equal(t, int64(1), req.ID)
	})

	t.Run("1.0 request rejected without option", func(t *testing.T) {
		_, err := DecodeRequest([]byte(req10))
		require.Error(t, err)
	})

	t.Run("1.0 notification with null id decodes", func(t *testing.T) {
		req, err := DecodeRequest([]byte(`{"method":"notify","params":[],"id":null}`), WithAllowJSONRPC1())
		require.NoError(t, err)
		assert.True(t, req.IsNotification())
	})
}

func TestWithAllowJSONRPC1_Batch(t *testing.T) {
	t.Run("batch response honors option per element", func(t *testing.T) {
		data := []byte(`[{"result":1,"error":null,"id":1},{"result":2,"error":null,"id":2}]`)
		resps, err := DecodeBatchResponse(data, WithAllowJSONRPC1())
		require.NoError(t, err)
		require.Len(t, resps, 2)

		_, err = DecodeBatchResponse(data)
		require.Error(t, err, "rejected without option")
	})

	t.Run("response-or-batch honors option", func(t *testing.T) {
		resps, isBatch, err := DecodeResponseOrBatch([]byte(`{"result":1,"error":null,"id":1}`), WithAllowJSONRPC1())
		require.NoError(t, err)
		assert.False(t, isBatch)
		require.Len(t, resps, 1)
	})

	t.Run("batch request honors option per element", func(t *testing.T) {
		data := []byte(`[{"method":"a","params":[],"id":1},{"method":"b","params":[],"id":2}]`)
		reqs, err := DecodeBatchRequest(data, WithAllowJSONRPC1())
		require.NoError(t, err)
		require.Len(t, reqs, 2)

		_, err = DecodeBatchRequest(data)
		require.Error(t, err, "rejected without option")
	})
}

func TestWithAllowJSONRPC1_RejectedVersions(t *testing.T) {
	// Versions outside {"", "1.0", "2.0"} stay rejected even with the option. In particular
	// 1.1 is not supported: only 1.0 and 2.0 are accepted on decode.
	cases := []string{
		`{"jsonrpc":"1.1","result":1,"error":null,"id":1}`,
		`{"jsonrpc":"2.1","result":1,"error":null,"id":1}`,
		`{"jsonrpc":"3.0","result":1,"id":1}`,
		`{"jsonrpc":"2","result":1,"id":1}`,
		`{"jsonrpc":"2.0.0","result":1,"id":1}`,
	}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			_, err := DecodeResponse([]byte(tc), WithAllowJSONRPC1())
			require.Error(t, err, "unsupported version must be rejected even with the option")

			_, err = DecodeRequest([]byte(`{"jsonrpc":"1.1","method":"m","id":1}`), WithAllowJSONRPC1())
			require.Error(t, err)
		})
	}
}

func TestWithAllowJSONRPC1_ResponseIDVariants(t *testing.T) {
	cases := []struct {
		name string
		data string
		want any
	}{
		{"string id", `{"result":1,"error":null,"id":"abc"}`, "abc"},
		{"int id", `{"result":1,"error":null,"id":42}`, int64(42)},
		{"fractional id", `{"result":1,"error":null,"id":1.5}`, 1.5},
		{"null id", `{"result":1,"error":null,"id":null}`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := DecodeResponse([]byte(tc.data), WithAllowJSONRPC1())
			require.NoError(t, err)
			assert.Equal(t, tc.want, resp.IDOrNil())
		})
	}
}

func TestWithAllowJSONRPC1_ResponseResultVariants(t *testing.T) {
	cases := []struct {
		name string
		data string
		want string
	}{
		{"number", `{"result":953931,"error":null,"id":1}`, "953931"},
		{"string", `{"result":"hello","error":null,"id":1}`, `"hello"`},
		{"bool", `{"result":true,"error":null,"id":1}`, "true"},
		{"array", `{"result":[1,2,3],"error":null,"id":1}`, "[1,2,3]"},
		{"object", `{"result":{"k":"v"},"error":null,"id":1}`, `{"k":"v"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := DecodeResponse([]byte(tc.data), WithAllowJSONRPC1())
			require.NoError(t, err)
			assert.Nil(t, resp.Err())
			assert.Equal(t, tc.want, string(resp.RawResult()))
		})
	}
}

func TestWithAllowJSONRPC1_RequestVariants(t *testing.T) {
	t.Run("string id with positional params", func(t *testing.T) {
		req, err := DecodeRequest([]byte(`{"method":"sum","params":[1,2],"id":"abc"}`), WithAllowJSONRPC1())
		require.NoError(t, err)
		assert.Equal(t, "abc", req.ID)
		assert.Equal(t, []any{float64(1), float64(2)}, req.Params)
	})

	t.Run("named (object) params", func(t *testing.T) {
		req, err := DecodeRequest([]byte(`{"method":"set","params":{"k":"v"},"id":1}`), WithAllowJSONRPC1())
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"k": "v"}, req.Params)
	})

	t.Run("fractional id", func(t *testing.T) {
		req, err := DecodeRequest([]byte(`{"method":"m","params":[],"id":1.5}`), WithAllowJSONRPC1())
		require.NoError(t, err)
		assert.Equal(t, 1.5, req.ID)
	})

	t.Run("no params", func(t *testing.T) {
		req, err := DecodeRequest([]byte(`{"method":"ping","id":1}`), WithAllowJSONRPC1())
		require.NoError(t, err)
		assert.Nil(t, req.Params)
	})
}

func TestWithAllowJSONRPC1_Reader(t *testing.T) {
	resp, err := DecodeResponseFromReader(strings.NewReader(`{"result":953931,"error":null,"id":1}`), 0, WithAllowJSONRPC1())
	require.NoError(t, err)
	var n int64
	require.NoError(t, resp.UnmarshalResult(&n))
	assert.Equal(t, int64(953931), n)
}
