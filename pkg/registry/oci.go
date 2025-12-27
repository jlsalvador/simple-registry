package registry

type DescriptorManifest struct {
	// This REQUIRED property contains the media type of the referenced content.
	// Values MUST comply with RFC 6838, including the naming requirements in
	// its section 4.2.
	MediaType string `json:"mediaType"`

	// This REQUIRED property is the digest of the targeted content, conforming
	// to the requirements outlined in Digests.
	// Retrieved content SHOULD be verified against this digest when consumed
	// via untrusted sources.
	Digest string `json:"digest"`

	// This REQUIRED property specifies the size, in bytes, of the raw content.
	// This property exists so that a client will have an expected size for the
	// content before processing.
	// If the length of the retrieved content does not match the specified
	// length, the content SHOULD NOT be trusted.
	Size int64 `json:"size"`

	// This OPTIONAL property specifies a list of URIs from which this object
	// MAY be downloaded.
	// Each entry MUST conform to RFC 3986.
	// Entries SHOULD use the http and https schemes, as defined in RFC 7230.
	Urls []string `json:"urls,omitempty"`

	// This OPTIONAL property contains arbitrary metadata for this descriptor.
	// This OPTIONAL property MUST use the annotation rules.
	Annotations map[string]string `json:"annotations,omitempty"`

	// This OPTIONAL property contains an embedded representation of the
	// referenced content.
	// Values MUST conform to the Base 64 encoding, as defined in RFC 4648.
	// The decoded data MUST be identical to the referenced content and SHOULD
	// be verified against the digest and size fields by content consumers.
	// See Embedded Content for when this is appropriate.
	Data *string `json:"data,omitempty"`

	// This OPTIONAL property contains the type of an artifact when the
	// descriptor points to an artifact.
	// This is the value of the config descriptor mediaType when the descriptor
	// references an image manifest.
	// If defined, the value MUST comply with RFC 6838, including the naming
	// requirements in its section 4.2, and MAY be registered with IANA.
	ArtifactType *string `json:"artifactType,omitempty"`

	// Descriptors pointing to application/vnd.oci.image.manifest.v1+json SHOULD
	// include the extended field platform,
	// see Image Index Property Descriptions for details.
	Platform *struct {
		Architecture string   `json:"architecture"`
		Os           string   `json:"os"`
		OsVersion    *string  `json:"os.version,omitempty"`
		OsFeatures   []string `json:"os.features"`
		Variant      *string  `json:"variant,omitempty"`
		Features     []string `json:"features"`
	} `json:"platform,omitempty"`
}

// https://github.com/opencontainers/image-spec/blob/v1.1.1/image-index.md#image-index-property-descriptions
type ImageIndexManifest struct {
	// This REQUIRED property specifies the image manifest schema version.
	// For this version of the specification, this MUST be 2 to ensure backward
	// compatibility with older versions of Docker.
	// The value of this field will not change.
	// This field MAY be removed in a future version of the specification.
	SchemaVersion int `json:"schemaVersion"`

	// This property SHOULD be used and remain compatible with earlier versions
	// of this specification and with other similar external formats.
	// When used, this field MUST contain the media type
	// application/vnd.oci.image.index.v1+json.
	// This field usage differs from the descriptor use of mediaType.
	MediaType string `json:"mediaType"`

	// This OPTIONAL property contains the type of an artifact when the manifest
	// is used for an artifact.
	// This MUST be set when config.mediaType is set to the empty value.
	// If defined, the value MUST comply with RFC 6838, including the naming
	// requirements in its section 4.2, and MAY be registered with IANA.
	// Implementations storing or copying image manifests MUST NOT error on
	// encountering an artifactType that is unknown to the implementation.
	ArtifactType *string `json:"artifactType,omitempty"`

	// This REQUIRED property contains a list of manifests for specific
	// platforms.
	// While this property MUST be present, the size of the array MAY be zero.
	Manifests []DescriptorManifest `json:"manifests"`

	// This OPTIONAL property specifies a descriptor of another manifest.
	// This value defines a weak association to a separate
	// Merkle Directed Acyclic Graph (DAG) structure,
	// and is used by the referrers API to include this manifest in the list of
	// responses for the subject digest.
	Subject *struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	} `json:"subject,omitempty"`

	// This OPTIONAL property contains arbitrary metadata for the image index.
	// This OPTIONAL property MUST use the annotation rules.
	Annotations map[string]string `json:"annotations,omitempty"`
}

func NewImageIndexManifest() ImageIndexManifest {
	return ImageIndexManifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.index.v1+json",
	}
}

// https://github.com/opencontainers/image-spec/blob/v1.1.1/manifest.md#image-manifest
type ImageManifest struct {
	// This REQUIRED property specifies the image manifest schema version.
	// For this version of the specification, this MUST be 2 to ensure backward
	// compatibility with older versions of Docker.
	// The value of this field will not change. This field MAY be removed in a
	// future version of the specification.
	SchemaVersion int `json:"schemaVersion"`

	// This property SHOULD be used and remain compatible with earlier versions
	// of this specification and with other similar external formats.
	// When used, this field MUST contain the media type
	// application/vnd.oci.image.manifest.v1+json.
	// This field usage differs from the descriptor use of mediaType.
	MediaType string `json:"mediaType"`

	// This REQUIRED property references a configuration object for a container,
	// by digest.
	Config struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	} `json:"config"`

	// Each item in the array MUST be a descriptor.
	// For portability, layers SHOULD have at least one entry.
	// See the guidance for an empty descriptor below, and
	// DescriptorEmptyJSON of the reference code.
	Layers []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	} `json:"layers"`

	// This OPTIONAL property specifies a descriptor of another manifest.
	// This value defines a weak association to a separate
	// Merkle Directed Acyclic Graph (DAG) structure, and is used by the
	// referrers API to include this manifest in the list of responses for the
	// subject digest.
	Subject *DescriptorManifest `json:"subject,omitempty"`

	// This OPTIONAL property contains arbitrary metadata for the image manifest.
	// This OPTIONAL property MUST use the annotation rules.
	Annotations map[string]string `json:"annotations,omitempty"`
}
