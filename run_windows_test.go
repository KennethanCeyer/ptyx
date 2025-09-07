//go:build windows

package ptyx

func isExpectedWaitErrorAfterPTYClose(err error) bool {
	return err == nil
}
