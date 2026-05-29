package upload

// PutOption configures uploads.
type PutOption func(*putOptions)

type putOptions struct {
	contentType   string
	contentLength int64
	useMultipart  bool
}

func applyPutOptions(opts []PutOption) putOptions {
	var o putOptions
	for _, fn := range opts {
		if fn != nil {
			fn(&o)
		}
	}
	return o
}

// WithContentType sets Content-Type.
func WithContentType(ct string) PutOption {
	return func(o *putOptions) { o.contentType = ct }
}

// WithContentLength sets known object size (enables streaming upload optimizations).
func WithContentLength(n int64) PutOption {
	return func(o *putOptions) { o.contentLength = n }
}

// WithMultipart forces multipart upload regardless of threshold.
func WithMultipart() PutOption {
	return func(o *putOptions) { o.useMultipart = true }
}

// GetOption configures downloads.
type GetOption func(*getOptions)

type getOptions struct {
	useMultipart bool
}

func applyGetOptions(opts []GetOption) getOptions {
	var o getOptions
	for _, fn := range opts {
		if fn != nil {
			fn(&o)
		}
	}
	return o
}

// GetWithMultipart forces ranged/multipart download.
func GetWithMultipart() GetOption {
	return func(o *getOptions) { o.useMultipart = true }
}

// ListOption configures list operations.
type ListOption func(*listOptions)

type listOptions struct {
	maxKeys            int
	continuationToken  string
	delimiter          string
}

func applyListOptions(opts []ListOption) listOptions {
	var o listOptions
	for _, fn := range opts {
		if fn != nil {
			fn(&o)
		}
	}
	return o
}

// WithMaxKeys limits keys per page.
func WithMaxKeys(n int) ListOption {
	return func(o *listOptions) { o.maxKeys = n }
}

// WithContinuationToken continues a previous list page.
func WithContinuationToken(token string) ListOption {
	return func(o *listOptions) { o.continuationToken = token }
}

// WithDelimiter sets folder delimiter (e.g. "/").
func WithDelimiter(d string) ListOption {
	return func(o *listOptions) { o.delimiter = d }
}
