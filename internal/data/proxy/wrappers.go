package proxy

import "io"

// Blobs upload

func (s *ProxyDataStorage) BlobsUploadCreate(repo string) (uuid string, err error) {
	if s.ds == nil {
		return "", ErrDataStorageNotInitialized
	}

	return s.ds.BlobsUploadCreate(repo)
}
func (s *ProxyDataStorage) BlobsUploadCancel(repo, uuid string) error {
	if s.ds == nil {
		return ErrDataStorageNotInitialized
	}

	return s.ds.BlobsUploadCancel(repo, uuid)
}
func (s *ProxyDataStorage) BlobsUploadWrite(repo, uuid string, r io.Reader, start int64) error {
	if s.ds == nil {
		return ErrDataStorageNotInitialized
	}

	return s.ds.BlobsUploadWrite(repo, uuid, r, start)
}
func (s *ProxyDataStorage) BlobsUploadCommit(repo, uuid, digest string) error {
	if s.ds == nil {
		return ErrDataStorageNotInitialized
	}

	return s.ds.BlobsUploadCommit(repo, uuid, digest)
}
func (s *ProxyDataStorage) BlobsUploadSize(repo, uuid string) (size int64, err error) {
	if s.ds == nil {
		return -1, ErrDataStorageNotInitialized
	}

	return s.ds.BlobsUploadSize(repo, uuid)
}

// Blobs

func (s *ProxyDataStorage) BlobsDelete(repo, digest string) error {
	if s.ds == nil {
		return ErrDataStorageNotInitialized
	}

	return s.ds.BlobsDelete(repo, digest)
}

// Manifests

func (s *ProxyDataStorage) ManifestPut(repo, reference string, r io.Reader) (dgst string, err error) {
	if s.ds == nil {
		return "", ErrDataStorageNotInitialized
	}

	return s.ds.ManifestPut(repo, reference, r)
}
func (s *ProxyDataStorage) ManifestDelete(repo, reference string) error {
	if s.ds == nil {
		return ErrDataStorageNotInitialized
	}

	return s.ds.ManifestDelete(repo, reference)
}

// Repositories

func (s *ProxyDataStorage) RepositoriesList() ([]string, error) {
	if s.ds == nil {
		return nil, ErrDataStorageNotInitialized
	}

	return s.ds.RepositoriesList()
}
