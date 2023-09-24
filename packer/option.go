package packer

type Option struct {
	URL         string
	Output      string
	Timeout     int
	ClutterFree bool
	Headers     map[string]string
}
