package IPCounter

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestCorrectOffset(t *testing.T) {
	tests := []struct {
		name           string
		lineBreak      byte
		initialOffset  int64
		buffer         []byte
		expectedOffset int64
	}{
		{
			name:           "Offset aligned with line break",
			lineBreak:      '\n',
			initialOffset:  0,
			buffer:         []byte("192.168.0.1\n"),
			expectedOffset: 12,
		},
		{
			name:           "Offset not aligned with line break",
			lineBreak:      '\n',
			initialOffset:  0,
			buffer:         []byte("192.168.0.1"),
			expectedOffset: 0,
		},
		{
			name:           "Offset with multiple line breaks",
			lineBreak:      '\n',
			initialOffset:  10,
			buffer:         []byte("More data\nAnother line\n"),
			expectedOffset: 33,
		},
		{
			name:           "Offset with empty buffer",
			lineBreak:      '\n',
			initialOffset:  0,
			buffer:         []byte(""),
			expectedOffset: 0,
		},
		{
			name:           "Offset with no line break",
			lineBreak:      '\n',
			initialOffset:  5,
			buffer:         []byte("data without newline"),
			expectedOffset: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipCounter := &IPCounter{
				lineBreak: tt.lineBreak,
			}
			offset := tt.initialOffset + int64(len(tt.buffer))
			ipCounter.correctOffset(&offset, tt.buffer)
			if offset != tt.expectedOffset {
				t.Errorf("correctOffset() = %v, want %v", offset, tt.expectedOffset)
			}
		})
	}
}

func TestGetPositions(t *testing.T) {
	content := []byte("192.168.0.1\n192.168.0.2\n192.168.0.3\n\n\n192.168.0.3\n\n\n192.168.0.3\n\n192.168.0.3\n")
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // Удаление файла после теста

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	fileSize, err := getFileSize(tmpFile)
	if err != nil {
		t.Fatalf("Failed to get file size: %v", err)
	}

	// Инициализация IPCounter
	counter := &IPCounter{
		file:          tmpFile,
		fileSize:      fileSize,
		maxGoroutines: 2,
		lineBreak:     '\n',
	}
	positions, err := counter.getPositions(context.Background(), maxLengthIp4)
	if err != nil {
		t.Errorf("getPositions returned error: %v", err)
	}

	expectedPositions := []int64{52, fileSize} //Estimated position values
	for i, pos := range positions {
		if pos != expectedPositions[i] {
			t.Errorf("Expected position %d, but received %d", expectedPositions[i], pos)
		}
	}

	// Закрываем файл после теста
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close file: %v", err)
	}
}

func TestIP4SequentialReader(t *testing.T) {
	const largeCount = 207
	tests := []struct {
		name          string
		inputData     string
		expectedCount int64
		lineBreak     byte
	}{
		{
			name:          "Empty File",
			inputData:     "",
			expectedCount: 0,
			lineBreak:     '\n',
		},
		{
			name:          "Single IP",
			inputData:     "192.168.1.1\n",
			expectedCount: 1,
			lineBreak:     '\n',
		},
		{
			name:          "Duplicate IPs",
			inputData:     "192.168.1.1\n192.168.1.1\n",
			expectedCount: 1,
			lineBreak:     '\n',
		},
		{
			name:          "Multiple IPs",
			inputData:     "192.168.1.1\n10.0.0.1\n172.16.0.1\n",
			expectedCount: 3,
			lineBreak:     '\n',
		},
		{
			name:          "IP with Extra Spaces",
			inputData:     "192.168.1.1   \n   10.0.0.1\n172.16.0.1\n",
			expectedCount: 3,
			lineBreak:     '\n',
		},
		{
			name:          "File with Invalid IP",
			inputData:     "192.168.1.1\ninvalid_ip\n10.0.0.1\n",
			expectedCount: 2,
			lineBreak:     '\n',
		},
		{
			name:          "File with Missing Line Break",
			inputData:     "192.168.1.1\n10.0.0.1\n172.16.0.1",
			expectedCount: 3,
			lineBreak:     '\n',
		},
		{
			name:          "File with Corrupted Data",
			inputData:     "192.168.1.1\n10.0.0.1\n172.16.0.1\ncorrupted_data",
			expectedCount: 3,
			lineBreak:     '\n',
		},
		{
			name:          "File with Multiple Line Breaks",
			inputData:     "192.168.1.1\n\n10.0.0.1\n\n172.16.0.1\n",
			expectedCount: 3,
			lineBreak:     '\n',
		},
		{
			name:          "Large File With New Line",
			inputData:     generateLargeFileData(100000, '\n'), // Function to generate large amount of IPs
			expectedCount: 255,
			lineBreak:     '\n',
		},
		{
			name:          "Large File With New Comma",
			inputData:     generateLargeFileData(100000, ','), // Function to generate large amount of IPs
			expectedCount: 255,
			lineBreak:     ',',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpFile, err := os.CreateTemp("", "testfile")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write test data to the file
			if _, err := tmpFile.WriteString(tt.inputData); err != nil {
				t.Fatalf("Failed to write data to temp file: %v", err)
			}

			// Close the file and reopen it to ensure data is flushed
			if err := tmpFile.Close(); err != nil {
				t.Fatalf("Failed to close temp file: %v", err)
			}

			// Create an IPCounter instance
			ipCounter := NewIPCounter(1, tt.lineBreak)

			// Call ip4SequentialReader
			file, err := os.Open(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to open temp file: %v", err)
			}
			defer file.Close()

			ipCounter.file = file
			ipCounter.fileSize, err = getFileSize(file)
			if err != nil {
				t.Fatalf("can't get file size: %v", err)
			}
			gotCount, err := ipCounter.ip4SequentialReader(context.Background())
			if err != nil {
				t.Fatalf("ip4SequentialReader returned an error: %v", err)
			}

			if gotCount != tt.expectedCount {
				t.Errorf("ip4SequentialReader() = %v; want %v", gotCount, tt.expectedCount)
			}
		})
	}
}

// Helper function to generate large file data
func generateLargeFileData(count int, linBreak byte) string {
	var result string
	str := string(linBreak)
	for i := 0; i < count; i++ {
		result += fmt.Sprintf("192.168.1.%d%s", i%255, str)
	}
	return result
}
