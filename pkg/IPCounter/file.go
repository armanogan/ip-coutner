package IPCounter

import "os"

type fileReadStat interface {
	Stat() (os.FileInfo, error)
}

func getFileSize(file fileReadStat) (int64, error) {
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
