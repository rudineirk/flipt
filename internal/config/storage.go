package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

// cheers up the unparam linter
var _ defaulter = (*StorageConfig)(nil)

type StorageType string

const (
	DatabaseStorageType = StorageType("database")
	LocalStorageType    = StorageType("local")
	GitStorageType      = StorageType("git")
	ObjectStorageType   = StorageType("object")
)

type ObjectSubStorageType string

const (
	S3ObjectSubStorageType = ObjectSubStorageType("s3")
)

// StorageConfig contains fields which will configure the type of backend in which Flipt will serve
// flag state.
type StorageConfig struct {
	Type     StorageType `json:"type,omitempty" mapstructure:"type" yaml:"type,omitempty"`
	Local    *Local      `json:"local,omitempty" mapstructure:"local,omitempty" yaml:"local,omitempty"`
	Git      *Git        `json:"git,omitempty" mapstructure:"git,omitempty" yaml:"git,omitempty"`
	Object   *Object     `json:"object,omitempty" mapstructure:"object,omitempty" yaml:"object,omitempty"`
	ReadOnly *bool       `json:"readOnly,omitempty" mapstructure:"readOnly,omitempty" yaml:"read_only,omitempty"`
}

func (c *StorageConfig) setDefaults(v *viper.Viper) error {
	switch v.GetString("storage.type") {
	case string(LocalStorageType):
		v.SetDefault("storage.local.path", ".")
	case string(GitStorageType):
		v.SetDefault("storage.git.ref", "main")
		v.SetDefault("storage.git.poll_interval", "30s")
	case string(ObjectStorageType):
		// keep this as a case statement in anticipation of
		// more object types in the future
		// nolint:gocritic
		switch v.GetString("storage.object.type") {
		case string(S3ObjectSubStorageType):
			v.SetDefault("storage.object.s3.poll_interval", "1m")
		}
	default:
		v.SetDefault("storage.type", "database")
	}

	return nil
}

func (c *StorageConfig) validate() error {
	switch c.Type {
	case GitStorageType:
		if c.Git.Ref == "" {
			return errors.New("git ref must be specified")
		}
		if c.Git.Repository == "" {
			return errors.New("git repository must be specified")
		}

		if err := c.Git.Authentication.validate(); err != nil {
			return err
		}

	case LocalStorageType:

		if c.Local.Path == "" {
			return errors.New("local path must be specified")
		}

	case ObjectStorageType:

		if c.Object == nil {
			return errors.New("object storage type must be specified")
		}
		if err := c.Object.validate(); err != nil {
			return err
		}
	}

	// setting read only mode is only supported with database storage
	if c.ReadOnly != nil && !*c.ReadOnly && c.Type != DatabaseStorageType {
		return errors.New("setting read only mode is only supported with database storage")
	}

	return nil
}

// Local contains configuration for referencing a local filesystem.
type Local struct {
	Path string `json:"path,omitempty" mapstructure:"path"`
}

// Git contains configuration for referencing a git repository.
type Git struct {
	Repository     string         `json:"repository,omitempty" mapstructure:"repository" yaml:"repository,omitempty"`
	Ref            string         `json:"ref,omitempty" mapstructure:"ref" yaml:"ref,omitempty"`
	PollInterval   time.Duration  `json:"pollInterval,omitempty" mapstructure:"poll_interval" yaml:"poll_interval,omitempty"`
	Authentication Authentication `json:"-" mapstructure:"authentication,omitempty" yaml:"-"`
}

// Object contains configuration of readonly object storage.
type Object struct {
	Type ObjectSubStorageType `json:"type,omitempty" mapstructure:"type" yaml:"type,omitempty"`
	S3   *S3                  `json:"s3,omitempty" mapstructure:"s3,omitempty" yaml:"s3,omitempty"`
}

// validate is only called if storage.type == "object"
func (o *Object) validate() error {
	switch o.Type {
	case S3ObjectSubStorageType:
		if o.S3 == nil || o.S3.Bucket == "" {
			return errors.New("s3 bucket must be specified")
		}
	default:
		return errors.New("object storage type must be specified")
	}
	return nil
}

// S3 contains configuration for referencing a s3 bucket
type S3 struct {
	Endpoint     string        `json:"endpoint,omitempty" mapstructure:"endpoint" yaml:"endpoint,omitempty"`
	Bucket       string        `json:"bucket,omitempty" mapstructure:"bucket" yaml:"bucket,omitempty"`
	Prefix       string        `json:"prefix,omitempty" mapstructure:"prefix" yaml:"prefix,omitempty"`
	Region       string        `json:"region,omitempty" mapstructure:"region" yaml:"region,omitempty"`
	PollInterval time.Duration `json:"pollInterval,omitempty" mapstructure:"poll_interval" yaml:"poll_interval,omitempty"`
}

// Authentication holds structures for various types of auth we support.
// Token auth will take priority over Basic auth if both are provided.
//
// To make things easier, if there are multiple inputs that a particular auth method needs, and
// not all inputs are given but only partially, we will return a validation error.
// (e.g. if username for basic auth is given, and token is also given a validation error will be returned)
type Authentication struct {
	BasicAuth *BasicAuth `json:"-" mapstructure:"basic,omitempty" yaml:"-"`
	TokenAuth *TokenAuth `json:"-" mapstructure:"token,omitempty" yaml:"-"`
}

func (a *Authentication) validate() error {
	if a.BasicAuth != nil {
		if err := a.BasicAuth.validate(); err != nil {
			return err
		}
	}
	if a.TokenAuth != nil {
		if err := a.TokenAuth.validate(); err != nil {
			return err
		}
	}

	return nil
}

// BasicAuth has configuration for authenticating with private git repositories
// with basic auth.
type BasicAuth struct {
	Username string `json:"-" mapstructure:"username" yaml:"-"`
	Password string `json:"-" mapstructure:"password" yaml:"-"`
}

func (b BasicAuth) validate() error {
	if (b.Username != "" && b.Password == "") || (b.Username == "" && b.Password != "") {
		return errors.New("both username and password need to be provided for basic auth")
	}

	return nil
}

// TokenAuth has configuration for authenticating with private git repositories
// with token auth.
type TokenAuth struct {
	AccessToken string `json:"-" mapstructure:"access_token" yaml:"-"`
}

func (t TokenAuth) validate() error { return nil }
