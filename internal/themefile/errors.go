package themefile

import "fmt"

// ResourceError says that the resource of a record runs past the end of the file.
type ResourceError struct {
	Slot     int
	TypeName string
	Field    string // the JSON name of the field of the offset, which is "offset" for every type
	Offset   uint32
	Size     uint64
	FileSize int
}

func (resourceError *ResourceError) Error() string {
	return fmt.Sprintf("slot %d (%s): %s 0x%x with %d bytes runs past the end of the file of %d bytes",
		resourceError.Slot, resourceError.TypeName, resourceError.Field, resourceError.Offset, resourceError.Size, resourceError.FileSize)
}
