package db

import (
	"context"
	"errors"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
)

// holds string (filepath) versions of lookupKey
type fsLookupKey struct {
	Default string
	Translation string
}

// pure filesystem backend implementation if the Db interface.
type fsDb struct {
	baseDb
	dir string
}

// NewFsDb creates a filesystem backed Db implementation.
func NewFsDb() *fsDb {
	db := &fsDb{}
	db.baseDb.defaultLock()
	return db
}

// Connect implements the Db interface.
func(fdb *fsDb) Connect(ctx context.Context, connStr string) error {
	if fdb.dir != "" {
		panic("already connected")
	}
	err := os.MkdirAll(connStr, 0700)
	if err != nil {
		return err
	}
	fdb.dir = connStr
	return nil
}

// Get implements the Db interface.
func(fdb *fsDb) Get(ctx context.Context, key []byte) ([]byte, error) {
	var f *os.File
	lk, err := fdb.ToKey(ctx, key)
	if err != nil {
		return nil, err
	}
	flk, err := fdb.pathFor(ctx, &lk)
	if err != nil {
		return nil, err
	}
	flka, err := fdb.altPathFor(ctx, &lk)
	if err != nil {
		return nil, err
	}
	for i, fp := range([]string{flk.Translation, flka.Translation, flk.Default, flka.Default}) {
		if fp == "" {
			logg.TraceCtxf(ctx, "fs get skip missing", "i", i)
			continue
		}
		logg.TraceCtxf(ctx, "trying fs get", "i", i, "key", key, "path", fp)
		f, err = os.Open(fp)
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if f == nil {
		return nil, NewErrNotFound(key)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Put implements the Db interface.
func(fdb *fsDb) Put(ctx context.Context, key []byte, val []byte) error {
	if !fdb.checkPut() {
		return errors.New("unsafe put and safety set")
	}
	lk, err := fdb.ToKey(ctx, key)
	if err != nil {
		return err
	}
	flk, err := fdb.pathFor(ctx, &lk)
	if err != nil {
		return err
	}
	if flk.Translation != "" {
		err = ioutil.WriteFile(flk.Translation, val, 0600)
		if err != nil {
			return err
		}
	}
	return ioutil.WriteFile(flk.Default, val, 0600)
}

// Close implements the Db interface.
func(fdb *fsDb) Close() error {
	return nil
}

// create a key safe for the filesystem.
func(fdb *fsDb) pathFor(ctx context.Context, lk *lookupKey) (fsLookupKey, error) {
	var flk fsLookupKey
	lk.Default[0] += 0x30
	flk.Default = path.Join(fdb.dir, string(lk.Default))
	if lk.Translation != nil {
		lk.Translation[0] += 0x30
		flk.Translation = path.Join(fdb.dir, string(lk.Translation))
	}
	return flk, nil
}

// create a key safe for the filesystem, matching legacy resource.FsResource name.
func(fdb *fsDb) altPathFor(ctx context.Context, lk *lookupKey) (fsLookupKey, error) {
	var flk fsLookupKey
	fb := string(lk.Default[1:])
	if fdb.pfx == DATATYPE_BIN {
		fb += ".bin"
	}
	flk.Default = path.Join(fdb.dir, fb)

	if lk.Translation != nil {
		fb = string(lk.Translation[1:])
		if fdb.pfx == DATATYPE_BIN {
			fb += ".bin"
		}
		flk.Translation = path.Join(fdb.dir, fb)
	}

	return flk, nil
}
