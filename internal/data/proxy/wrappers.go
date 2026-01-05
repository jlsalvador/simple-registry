package proxy

import (
	"io"
	"iter"
	"time"
)

// Blobs upload

func (s *ProxyDataStorage) BlobsUploadCreate(repo string) (uuid string, err error) {
	if s.Next == nil {
		return "", ErrDataStorageNotInitialized
	}

	return s.Next.BlobsUploadCreate(repo)
}
func (s *ProxyDataStorage) BlobsUploadCancel(repo, uuid string) error {
	if s.Next == nil {
		return ErrDataStorageNotInitialized
	}

	return s.Next.BlobsUploadCancel(repo, uuid)
}
func (s *ProxyDataStorage) BlobsUploadWrite(repo, uuid string, r io.Reader, start int64) error {
	if s.Next == nil {
		return ErrDataStorageNotInitialized
	}

	return s.Next.BlobsUploadWrite(repo, uuid, r, start)
}
func (s *ProxyDataStorage) BlobsUploadCommit(repo, uuid, digest string) error {
	if s.Next == nil {
		return ErrDataStorageNotInitialized
	}

	return s.Next.BlobsUploadCommit(repo, uuid, digest)
}
func (s *ProxyDataStorage) BlobsUploadSize(repo, uuid string) (size int64, err error) {
	if s.Next == nil {
		return -1, ErrDataStorageNotInitialized
	}

	return s.Next.BlobsUploadSize(repo, uuid)
}

// Blobs

func (s *ProxyDataStorage) BlobsDelete(repo, digest string) error {
	if s.Next == nil {
		return ErrDataStorageNotInitialized
	}

	return s.Next.BlobsDelete(repo, digest)
}
func (s *ProxyDataStorage) BlobsList() (digests iter.Seq[string], err error) {
	if s.Next == nil {
		return nil, ErrDataStorageNotInitialized
	}

	return s.Next.BlobsList()
}
func (s *ProxyDataStorage) BlobLastAccess(digest string) (lastAccess time.Time, err error) {
	if s.Next == nil {
		return time.Now(), ErrDataStorageNotInitialized
	}

	return s.Next.BlobLastAccess(digest)
}

// Manifests

func (s *ProxyDataStorage) ManifestPut(repo, reference string, r io.Reader) (dgst string, err error) {
	if s.Next == nil {
		return "", ErrDataStorageNotInitialized
	}

	return s.Next.ManifestPut(repo, reference, r)
}
func (s *ProxyDataStorage) ManifestDelete(repo, reference string) error {
	if s.Next == nil {
		return ErrDataStorageNotInitialized
	}

	return s.Next.ManifestDelete(repo, reference)
}
func (s *ProxyDataStorage) ManifestsList(repo string) (digests iter.Seq[string], err error) {
	if s.Next == nil {
		return nil, ErrDataStorageNotInitialized
	}

	return s.Next.ManifestsList(repo)
}
func (s *ProxyDataStorage) ManifestLastAccess(digest string) (lastAccess time.Time, err error) {
	if s.Next == nil {
		return time.Now(), ErrDataStorageNotInitialized
	}

	return s.Next.BlobLastAccess(digest)
}

// Repositories

func (s *ProxyDataStorage) RepositoriesList() ([]string, error) {
	if s.Next == nil {
		return nil, ErrDataStorageNotInitialized
	}

	return s.Next.RepositoriesList()
}
