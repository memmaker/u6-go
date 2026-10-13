package game

// Install unpacks MAP from LZMAP into dir when it is missing, and creates the saved game when
// it is missing, as INSTALL.EXE did when copying the original disks to a hard disk.
func Install(dir string) error { return ensureGameFiles(dir) }
