// +build !go1.13

package langserver

func succeeded(err error) bool {
	return err == nil
}
