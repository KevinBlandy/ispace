package types

type DataSize uint64

var (
	Byte = DataSize(1)
	KB   = DataSize(Byte * 1024)
	MB   = DataSize(KB * 1024)
	GB   = DataSize(MB * 1024)
	TB   = DataSize(GB * 1024)
	PB   = DataSize(TB * 1024)
	EB   = DataSize(PB * 1024)
)
