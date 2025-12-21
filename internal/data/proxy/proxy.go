package proxy

import (
	"errors"
	"io"
	"io/fs"
)

// Blobs

func (s *ProxyDataStorage) BlobsGet(repo, digest string) (r io.ReadCloser, size int64, err error) {
	if s.ds == nil {
		return nil, -1, ErrDataStorageNotInitialized
	}

	// 1. Try local
	r, size, err = s.ds.BlobsGet(repo, digest)
	if err == nil {
		return r, size, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, -1, err
	}

	// 2. Proxy?
	proxy := s.matchProxy(repo)
	if proxy == nil {
		return nil, -1, fs.ErrNotExist
	}

	// 3. Fetch upstream
	upstreamReader, size, err := fetchBlobFromUpstream(proxy, repo, digest)
	if err != nil {
		return nil, -1, err
	}
	defer upstreamReader.Close()

	// 4. Cache locally
	uuid, err := s.ds.BlobsUploadCreate(repo)
	if err != nil {
		return nil, -1, err
	}
	if err := s.ds.BlobsUploadWrite(repo, uuid, upstreamReader, -1); err != nil {
		return nil, -1, err
	}
	if err := s.ds.BlobsUploadCommit(repo, uuid, digest); err != nil {
		return nil, -1, err
	}

	// 5. Return locally cached blob
	return s.ds.BlobsGet(repo, digest)
}

// Manifests

func (s *ProxyDataStorage) ManifestGet(repo, reference string) (r io.ReadCloser, size int64, err error) {
	if s.ds == nil {
		return nil, -1, ErrDataStorageNotInitialized
	}

	// 1. Try local
	r, size, err = s.ds.ManifestGet(repo, reference)
	if err == nil {
		return r, size, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, -1, err
	}

	// 2. Proxy?
	proxy := s.matchProxy(repo)
	if proxy == nil {
		return nil, -1, fs.ErrNotExist
	}

	// 3. Fetch upstream
	upstreamReader, size, err := fetchManifestFromUpstream(proxy, repo, reference)
	if err != nil {
		return nil, -1, err
	}
	defer upstreamReader.Close()

	// 4. Cache upstream manifest locally
	if _, err := s.ds.ManifestPut(repo, reference, upstreamReader); err != nil {
		return nil, -1, err
	}

	return s.ds.ManifestGet(repo, reference)
}

// Referrers

func (s *ProxyDataStorage) ReferrersGet(repo, dgst string) (r io.ReadCloser, size int64, err error) {
	if s.ds == nil {
		return nil, -1, ErrDataStorageNotInitialized
	}

	// 1. Try local first
	r, size, err = s.ds.ReferrersGet(repo, dgst)
	if err == nil {
		return r, size, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, -1, err
	}

	// 2. Find matching proxy
	proxy := s.matchProxy(repo)
	if proxy == nil {
		return nil, -1, fs.ErrNotExist
	}

	// 3. Fetch from upstream
	body, size, err := fetchReferrersFromUpstream(*proxy, repo, dgst)
	if err != nil {
		return nil, -1, err
	}
	defer body.Close()

	// 4. Store locally
	dgstRef := "sha256:" + dgst // referrers are indexed by subject digest
	if _, err := s.ds.ManifestPut(repo, dgstRef, body); err != nil {
		return nil, -1, err
	}

	// 5. Read back from local (canonical)
	return s.ds.ReferrersGet(repo, dgst)
}

// Tags

func (s *ProxyDataStorage) TagsList(repo string) ([]string, error) {
	if s.ds == nil {
		return nil, ErrDataStorageNotInitialized
	}

	// 1. Try local first
	tags, err := s.ds.TagsList(repo)
	if err == nil {
		return tags, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	// 2. Try upstream
	proxy := s.matchProxy(repo)
	if proxy == nil {
		return nil, fs.ErrNotExist
	}

	return fetchTagsFromUpstream(proxy, repo)
}
