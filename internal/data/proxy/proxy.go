package proxy

import (
	"errors"
	"io"
	"io/fs"
	"regexp"

	"github.com/jlsalvador/simple-registry/pkg/registry"
)

// Blobs

func (s *ProxyDataStorage) BlobsGet(repo, digest string) (r io.ReadCloser, size int64, err error) {
	if s.ds == nil {
		return nil, -1, ErrDataStorageNotInitialized
	}

	// Try local first.
	r, size, err = s.ds.BlobsGet(repo, digest)
	if err == nil {
		return r, size, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, -1, err
	}

	// Find matching proxy.
	proxy := s.matchProxy(repo)
	if proxy == nil {
		return nil, -1, fs.ErrNotExist
	}

	// Fetch from upstream.
	upstreamReader, size, err := fetchBlobFromUpstream(proxy, repo, digest)
	if err != nil {
		return nil, -1, err
	}
	defer upstreamReader.Close()

	// Store locally.
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

	// Read back from local.
	return s.ds.BlobsGet(repo, digest)
}

// Manifests

func (s *ProxyDataStorage) ManifestGet(repo, reference string) (
	r io.ReadCloser,
	size int64,
	digest string,
	err error,
) {
	if s.ds == nil {
		return nil, -1, "", ErrDataStorageNotInitialized
	}

	proxy := s.matchProxy(repo)
	isDigest := regexp.MustCompile("^" + registry.RegExpDigest + "$").MatchString(reference)
	isTag := regexp.MustCompile("^" + registry.RegExpTag + "$").MatchString(reference)

	// If Proxy is found and the reference is a tag,
	// update manifest from the upstream.
	var upstreamDigest string
	if proxy != nil && !isDigest && isTag {

		// Fetch lastest digest for this tag from upstream.
		upstreamDigest, _ = fetchManifestDigestHEAD(proxy, repo, reference)
	}

	// Try to get from local.
	r, size, digest, err = s.ds.ManifestGet(repo, reference)
	if err == nil {
		if upstreamDigest == "" || upstreamDigest == digest {
			return r, size, digest, nil
		}

		_ = r.Close()
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, -1, "", err
	}

	// Local miss, check Proxy.
	if proxy == nil {
		return nil, -1, "", fs.ErrNotExist
	}

	// Fetch from upstream.
	upstreamReader, size, err := fetchManifestFromUpstream(proxy, repo, reference)
	if err != nil {
		return nil, -1, "", err
	}
	defer upstreamReader.Close()

	// Store locally.
	newDigest, err := s.ds.ManifestPut(repo, reference, upstreamReader)
	if err != nil {
		return nil, -1, "", err
	}

	// Read back from local.
	return s.ds.ManifestGet(repo, newDigest)
}

// Referrers

func (s *ProxyDataStorage) ReferrersGet(repo, dgst string) (r io.ReadCloser, size int64, err error) {
	if s.ds == nil {
		return nil, -1, ErrDataStorageNotInitialized
	}

	// Try local first.
	r, size, err = s.ds.ReferrersGet(repo, dgst)
	if err == nil {
		return r, size, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, -1, err
	}

	// Find matching proxy.
	proxy := s.matchProxy(repo)
	if proxy == nil {
		return nil, -1, fs.ErrNotExist
	}

	// Fetch from upstream.
	body, size, err := fetchReferrersFromUpstream(*proxy, repo, dgst)
	if err != nil {
		return nil, -1, err
	}
	defer body.Close()

	// Store locally.
	dgstRef := "sha256:" + dgst // referrers are indexed by subject digest
	if _, err := s.ds.ManifestPut(repo, dgstRef, body); err != nil {
		return nil, -1, err
	}

	// Read back from local.
	return s.ds.ReferrersGet(repo, dgst)
}

// Tags

func (s *ProxyDataStorage) TagsList(repo string) ([]string, error) {
	if s.ds == nil {
		return nil, ErrDataStorageNotInitialized
	}

	proxy := s.matchProxy(repo)

	if proxy != nil {
		tags, err := fetchTagsFromUpstream(proxy, repo)
		if err == nil {
			return tags, nil
		}
		// upstream failed, fallback to local.
	}

	return s.ds.TagsList(repo)
}
