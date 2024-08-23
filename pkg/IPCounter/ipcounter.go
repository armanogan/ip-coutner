package IPCounter

import (
	"bufio"
	"context"
	"errors"
	"golang.org/x/sync/errgroup"
	"io"
	"net"
	"os"
	"runtime"
	"time"
)

const maxLengthIp4 = net.IPv4len * 4 //16 is the maximum size for ip4(255.255.255.255) + 1 byte lineBreak

type fileReader interface {
	Read(p []byte) (n int, err error)
	ReadAt(p []byte, off int64) (n int, err error)
}

type IPCounter struct {
	file          fileReader
	fileSize      int64
	maxGoroutines int64
	lineBreak     byte
}

func NewIPCounter(maxGoroutines int64, lineBreak byte) *IPCounter {
	return &IPCounter{maxGoroutines: maxGoroutines, lineBreak: lineBreak, fileSize: 0, file: nil}
}

func (ip *IPCounter) GetMaxGoroutines() int64 {
	return ip.maxGoroutines
}

func (ip *IPCounter) GetFileSize() int64 {
	return ip.fileSize
}

// correctOffset adjusts the offset to ensure it aligns with the line break character.
func (ip *IPCounter) correctOffset(offset *int64, buffer []byte) {
	if len(buffer) > 0 && buffer[len(buffer)-1] != ip.lineBreak {
		for j := len(buffer) - 1; j > -1; j-- {
			if buffer[j] == ip.lineBreak {
				break
			}
			*offset--
		}
	}
}

func (ip *IPCounter) UniqueIP4(ctx context.Context, path string) (int64, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	file, _err := os.Open(path) // Can another process write it, do we need the shared flock?
	if _err != nil {
		return 0, _err
	}
	var (
		err         error
		uniqueCount int64
	)
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	size, sizeErr := getFileSize(file)
	if sizeErr != nil {
		return 0, sizeErr
	}
	ip.file = file
	ip.fileSize = size
	ip.maxGoroutines = ip.getGoroutinesCount()
	if ip.maxGoroutines > 1 { //use goroutines
		uniqueCount, err = ip.ip4multipleReaders(ctx)
	} else {
		uniqueCount, err = ip.ip4SequentialReader(ctx)
	}
	return uniqueCount, err
}

// UniqueIP6 is a placeholder for processing IPv6 addresses, to be implemented similarly to IPv4.
func (ip *IPCounter) UniqueIP6(ctx context.Context, path string) (int64, error) {
	return 0, nil
}

// ip4SequentialReader reads IPv4 addresses sequentially from the file and counts unique addresses.
func (ip *IPCounter) ip4SequentialReader(ctx context.Context) (int64, error) {
	const mb512 int64 = (256 * 256 * 256 * 256) / 8 //512mb
	var (
		capacity        int64
		uniqueCount     int64
		ipsWithIndex    []byte
		ipsWithOutIndex []uint32
		line            []byte
		err             error
	)
	if ip.fileSize <= mb512 {
		capacity = ip.fileSize / 8 //min length of ip is 7 bytes + new line
		if capacity == 0 {
			capacity = ip.fileSize
		}
		ipsWithOutIndex = make([]uint32, 0, capacity)
	} else {
		ipsWithIndex = make([]byte, mb512)
	}
	chunkSize := int(ip.fileSize)
	if ip.fileSize > 65536 {
		chunkSize = 65536
	}
	reader := bufio.NewReaderSize(ip.file, chunkSize)
	for {
		if checkContext(ctx) != nil {
			return 0, ctx.Err()
		}
		for j := 0; j < 3; j++ { //Retry reading up to 3 times
			line, err = reader.ReadBytes(ip.lineBreak)
			if err == nil || errors.Is(err, io.EOF) {
				break
			}
			if j == 2 {
				return 0, err
			}
			time.Sleep(10 * time.Millisecond) // Retry delay
		}
		if len(line) > 0 {
			if line[len(line)-1] == ip.lineBreak {
				line = line[:len(line)-1]
			}
			ip32, err2 := ip4BytesToUint32(line)
			if err2 == nil { //should be logged?
				if capacity == 0 {
					index := ip32 >> 3    //bitwise 3 shift right is the same as divide 8 without remainder(/8)
					byteIndex := ip32 & 7 //the same as divide 8 with remainder(%8)
					if ipsWithIndex[index]&(1<<byteIndex) == 0 {
						ipsWithIndex[index] |= 1 << byteIndex
						uniqueCount++
					}
				} else {
					ipsWithOutIndex = append(ipsWithOutIndex, ip32)
				}
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	if len(ipsWithOutIndex) > 0 {
		uniqueCount = ip4SortCount(ipsWithOutIndex)
	}
	return uniqueCount, nil
}

// ip4multipleReaders uses multiple goroutines to read IPv4 addresses in parallel and counts unique addresses.
func (ip *IPCounter) ip4multipleReaders(ctx context.Context) (int64, error) {
	positions, err := ip.getPositions(ctx, maxLengthIp4)
	if err != nil {
		return 0, err
	}
	var (
		ips         [(256 * 256 * 256 * 256) / 8]byte //512mb
		uniqueCount int64
	)
	ch := make(chan []byte, 255*255) //todo need to understand what size should have buffer
	errsGroup, erCtx := errgroup.WithContext(ctx)
	errsGroup.SetLimit(len(positions))

	for i := 0; i < len(positions); i++ {
		i := i
		var offset int64
		if i > 0 {
			offset = positions[i-1]
		}
		errsGroup.Go(func() error {
			return ip.goroutineReader(erCtx, offset, positions[i], ch, maxLengthIp4)
		})
	}

	go func() {
		err = errsGroup.Wait()
		close(ch)
	}()

	for v := range ch {
		ip32, err2 := ip4BytesToUint32(v)
		if err2 != nil { //should be logged?
			continue
		}
		index := ip32 >> 3    //bitwise 3 shift right is the same as divide 8 without remainder(/8)
		byteIndex := ip32 & 7 //the same as divide 8 with remainder(%8)
		if ips[index]&(1<<byteIndex) == 0 {
			ips[index] |= 1 << byteIndex
			uniqueCount++
		}
	}
	return uniqueCount, err
}

// goroutineReader reads bytes from the file in a goroutine and sends IPv4 addresses to the channel.
func (ip *IPCounter) goroutineReader(ctx context.Context, offset int64, limit int64, ch chan []byte, wordMaxLen64 int64) error {
	maxBytes := limit - offset
	chunkSize := maxBytes
	if maxBytes > 65536 {
		chunkSize = 65536
	}
	buffer := make([]byte, chunkSize)
	str := make([]byte, wordMaxLen64)
	var (
		maxSize   int64
		i         int64
		k         int64
		err       error
		bytesRead int
	)

	for maxBytes > 0 {
		if checkContext(ctx) != nil {
			return ctx.Err()
		}
		for j := 0; j < 3; j++ { // Retry reading up to 3 times
			bytesRead, err = ip.file.ReadAt(buffer, offset)
			if err == nil || errors.Is(err, io.EOF) {
				break
			}
			if j == 2 {
				return err
			}
			time.Sleep(10 * time.Millisecond) // Retry delay
		}

		offset += chunkSize
		beforeOffset := offset
		ip.correctOffset(&offset, buffer)
		bytesRead64 := int64(bytesRead)
		if bytesRead64 < chunkSize {
			maxSize = bytesRead64
		} else {
			maxSize = bytesRead64 - (beforeOffset - offset)
			if maxSize <= 0 {
				maxSize = bytesRead64
			}
		}
		for i = 0; i < maxSize; i++ {
			if buffer[i] == ip.lineBreak || i == maxSize-1 {
				if k > 0 {
					ch <- append([]byte(nil), str[:k]...)
					k = 0
				}
			} else if buffer[i] != ' ' {
				str[k] = buffer[i]
				k++
			}
			maxBytes--
			if maxBytes == 0 {
				break
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return nil
}

// getGoroutinesCount calculates the optimal number of goroutines based on file size and system resources.
func (ip *IPCounter) getGoroutinesCount() int64 {
	var chunkSize int64 = 1073741824 //1gb
	fileSize := ip.fileSize
	if fileSize < chunkSize {
		chunkSize = fileSize
	} else if fileSize <= 10737418240 { //10gb
		chunkSize = 104857600 //100mb
	}
	maxGoroutineNeed := fileSize / chunkSize
	if maxGoroutineNeed == 0 {
		maxGoroutineNeed = 1
	}
	maxGoroutine := ip.GetMaxGoroutines()
	if maxGoroutineNeed > maxGoroutine && maxGoroutine > 0 {
		maxGoroutineNeed = maxGoroutine
	}
	maxCores := int64(float64(runtime.GOMAXPROCS(0)) * 1.5) //150% of cores, but only 100% work parallel, others 50% work concurrently
	if maxGoroutineNeed > maxCores {
		maxGoroutineNeed = maxCores
	}
	return maxGoroutineNeed
}

// getPositions determines file offsets for each goroutine to process.
func (ip *IPCounter) getPositions(ctx context.Context, wordMaxLen uint) ([]int64, error) {
	var (
		err    error
		offset int64
		i      int64
	)
	maxGoroutine := ip.GetMaxGoroutines()
	file := ip.file
	fileSize := ip.fileSize
	chunkSize := fileSize / maxGoroutine
	buffer := make([]byte, wordMaxLen)
	offsets := make([]int64, 0, maxGoroutine)
	for i = 0; i < maxGoroutine; i++ {
		if err = checkContext(ctx); err != nil {
			return nil, err
		}

		offset += chunkSize
		if i == maxGoroutine-1 || offset > fileSize {
			offset = fileSize
		}
		for k := 0; k < 3; k++ { // Retry reading up to 3 times
			_, err = file.ReadAt(buffer, offset)
			if err == nil || errors.Is(err, io.EOF) {
				break
			}
			if k == 2 {
				return nil, err
			}
			time.Sleep(20 * time.Millisecond) // Retry delay
		}
		isEndFile := errors.Is(err, io.EOF)
		if !isEndFile {
			offset += int64(wordMaxLen)
			ip.correctOffset(&offset, buffer)
		}
		offsets = append(offsets, offset)

		if isEndFile {
			err = nil
			break
		}
	}
	return offsets, err
}
