package jsonrpc

// decodeOptions holds configuration applied during decode operations.
type decodeOptions struct {
	allowJSONRPC1 bool
}

// DecodeOption configures decoding behavior. Pass options to decode entry points
// such as DecodeRequest, DecodeResponse, and their batch variants. Options only
// affect decoding; the encode path always emits strict JSON-RPC 2.0.
type DecodeOption func(*decodeOptions)

// WithAllowJSONRPC1 relaxes version validation on decode to accept JSON-RPC 1.0
// payloads, which carry no "jsonrpc" field (or an explicit "jsonrpc":"1.0") and may
// include both result and a null error. The decoded version string is preserved as-is.
// Without this option, decoding stays strict JSON-RPC 2.0.
func WithAllowJSONRPC1() DecodeOption {
	return func(o *decodeOptions) {
		o.allowJSONRPC1 = true
	}
}

// applyDecodeOptions folds a list of options into a single decodeOptions value.
func applyDecodeOptions(opts []DecodeOption) decodeOptions {
	var o decodeOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// versionAllowedOnDecode reports whether a decoded jsonrpc version string is acceptable.
// Strict mode accepts only "2.0"; with allowJSONRPC1 it also accepts "" and "1.0".
func versionAllowedOnDecode(v string, allowJSONRPC1 bool) bool {
	if v == jsonRPCVersion {
		return true
	}
	return allowJSONRPC1 && (v == "" || v == "1.0")
}
