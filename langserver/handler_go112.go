// +build !go1.13

package langserver

func succeeded(err error) bool {
	if err == nil {
		return true
	}
	return false
}
