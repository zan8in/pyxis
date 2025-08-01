package pyxis

const (
	DefaultRetries   = 1
	DefaultTimeout   = 10
	DefaultRateLimit = 20 // 从150降低到50，更保守的默认值

	HostTempFile = "pyxis-host-temp-*"

	HTTP_PREFIX  = "http://"
	HTTPS_PREFIX = "https://"
)
