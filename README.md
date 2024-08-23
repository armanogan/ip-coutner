
IPCounter is a Go package designed for efficiently processing and analyzing IP address data from files. 

It supports reading IPv4 addresses and can handle large files using both sequential and concurrent processing strategies.

Features:

    Read IPv4 addresses from a file.
    Support for large files using efficient memory management.
    Concurrent processing with goroutines for faster performance on large files.
    Flexible goroutine management based on file size.
    Automatically skips invalid IPv4 addresses during processing.

Maximum RAM Usage:

The maximum RAM that can be used by the program is dependent on the file size, with a cap of 512MB. This ensures efficient memory usage even when processing large files.

Max Goroutines:

The maxGoroutines parameter controls the level of concurrency used during processing. Here’s how it works:

For small files (less than 10GB), the program processes the file sequentially without using goroutines, avoiding unnecessary overhead.

For larger files, the number of goroutines increases to parallelize reading and processing, which may improve performance. However, if the disk is an HDD or an SSD without parallel reading capabilities, it's better to set the maxGoroutines parameter to 1.

The number of goroutines is capped by the maxGoroutines parameter set during initialization and also considers the number of available CPU cores.

Setting maxGoroutines to 0 will try to autodetect the optimal number of goroutines based on the file size.

SSD and Parallel Reading:

Some SSD models support parallel reading, which can enhance performance when multiple goroutines read from the disk simultaneously. If your SSD supports this feature, higher concurrency levels might improve processing speed.

Newline Support:

The current solution does not support IPv4 files with CRLF (\r\n) newline characters. Ensure that the file uses LF (\n) newlines for compatibility.