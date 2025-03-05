package transfer

import "testing"

/**
 * For each test case, send the sendPath under testDir to a tmp directory,
 * and compare it with the expectedPath under testDir.
 */
func TestZipReadWrite(t *testing.T) {
	testReadWrite(t, ReadZip, WriteZip)
}
