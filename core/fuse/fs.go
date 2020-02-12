package fuse

import (
	"io"
	"os"
	"time"
	"fmt"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"golang.org/x/net/context"
)

// Entry is an entry for a file or dir in the file system.
type Entry struct {
	Attr fuseops.InodeAttributes
	Dir  *Dir
	File *File
}

// FakeDataFS is a filesystem filled with fake data.
type FS struct {	
	entries map[fuseops.InodeID]Entry
	cache   *Cache
	root string
	rootDir *Dir
	fuseutil.NotImplementedFileSystem
}

// NewFakeDataFS creates a new filesystem.
func NewFileSystem(ctx context.Context, root string) (*FS, error) {
	fs := &FS {
		root:		root,
		cache:		newCache(ctx),
		entries:	make(map[fuseops.InodeID]Entry),
	}

	//V("create filesystem with seed %v, max size %v, %v files per dir\n", seed, maxSize, filesPerDir)

	d := NewDir(fs, root)
	fs.entries[fuseops.RootInodeID] = Entry{
		Dir: d,
		Attr: fuseops.InodeAttributes{
			//size
			//Nlink
			Atime:  time.Now(),
			Ctime:  time.Now(),
			Mtime:  time.Now(),
			Crtime: time.Now(),

			Uid: uint32(os.Getuid()),
			Gid: uint32(os.Getgid()),

			Mode: os.ModeDir | 0555,
		},
	}

	fs.rootDir = d
	return fs, nil
}

var rootAttributes = fuseops.InodeAttributes{
	Atime:  time.Now(),
	Ctime:  time.Now(),
	Mtime:  time.Now(),
	Crtime: time.Now(),

	Uid: uint32(os.Getuid()),
	Gid: uint32(os.Getgid()),

	Mode: os.ModeDir | 0555,
}

// GetInodeAttributes returns information about an inode.
func (f *FS) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	entry, ok := f.entries[op.Inode]
	if !ok {
		return fuse.ENOENT
	}

	op.Attributes = entry.Attr
	return nil
}

// OpenDir opens a directory.
func (f *FS) OpenDir(ctx context.Context, op *fuseops.OpenDirOp) error {
	return nil
}

// ReleaseDirHandle frees a handle returned by OpenDir.
func (f *FS) ReleaseDirHandle(context.Context, *fuseops.ReleaseDirHandleOp) error {
	return nil
}

// ForgetInode frees an inode.
func (f *FS) ForgetInode(context.Context, *fuseops.ForgetInodeOp) error {
	return nil
}

// ReadDir lists a directory.
func (f *FS) ReadDir(ctx context.Context, op *fuseops.ReadDirOp) error {
	entry, ok := f.entries[op.Inode]
	if !ok {
		return fuse.ENOENT
	}

	if entry.Dir == nil {
		return fuse.EIO
	}

	if int(op.Offset) > len(entry.Dir.entries) {
		return fuse.EIO
	}

	op.BytesRead = entry.Dir.ReadDir(op.Dst, int(op.Offset))
	return nil
}

// StatFS apparently necessary and probably allows statfs to return
func (f *FS) StatFS(ctx context.Context, op *fuseops.StatFSOp) error {
	return nil
}

// OpenFile determines if a file can be opened
func (f *FS) OpenFile(ctx context.Context, op *fuseops.OpenFileOp) error {
	// Allow opening any file.
	return nil
}

// LookUpInode returns information on an inode.
func (f *FS) LookUpInode(ctx context.Context, op *fuseops.LookUpInodeOp) error {
	entry, ok := f.entries[op.Parent]
	if !ok {
		return fuse.ENOENT
	}

	if entry.Dir == nil {
		return fuse.EIO
	}

	d := entry.Dir

	for _, entry := range d.entries {
		if op.Name == entry.Name {
			fsEntry, ok := f.entries[entry.Inode]
			if !ok {
				return fuse.EIO
			}

			op.Entry.Child = entry.Inode
			op.Entry.Attributes = fsEntry.Attr
		}
	}

	return nil
}

// ReadFile reads data from a file.
func (f *FS) ReadFile(ctx context.Context, op *fuseops.ReadFileOp) error {
	entry, ok := f.entries[op.Inode]
	if !ok {
		return fuse.ENOENT
	}

	if entry.File == nil {
		return fuse.EIO
	}

	rd, err := f.cache.Get(op.Inode, op.Offset)
	if err != nil {
		rd = ContinuousFileReader(entry.File, op.Offset)
	}

	n, err := io.ReadFull(rd, op.Dst)
	if err == io.ErrUnexpectedEOF {
		err = nil
	} else {
		f.cache.Put(op.Inode, op.Offset+int64(n), rd)
	}
	op.BytesRead = len(op.Dst)
	return err
}

func (f *FS) Entries() map[fuseops.InodeID]Entry {
	fmt.Printf("%p\n", &f.entries)

	return f.entries
}