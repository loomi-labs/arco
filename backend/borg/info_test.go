package borg

import (
	"context"
	"github.com/loomi-labs/arco/backend/borg/mocks"
	"github.com/loomi-labs/arco/backend/borg/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"
)

/*
TEST CASES - info.go

TestBorgInfo
* emptyRepo
* nonEmptyRepo

*/

type testBorgInfo struct {
	name      string
	cmdResult []byte
	result    *types.InfoResponse
	wantErr   bool
}

var emptyRepo = testBorgInfo{
	name: "Call info - empty repo",
	cmdResult: []byte(`
{
    "cache": {
        "path": "/home/test/.cache/borg/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
        "stats": {
            "total_chunks": 0,
            "total_csize": 0,
            "total_size": 0,
            "total_unique_chunks": 0,
            "unique_csize": 0,
            "unique_size": 0
        }
    },
    "encryption": {
        "mode": "repokey-blake2"
    },
    "repository": {
        "id": "01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
        "last_modified": "2024-12-02T10:28:45.000000",
        "location": "/home/test/arcotest/dest/encrypted_with_123"
    },
    "security_dir": "/home/test/.config/borg/security/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf"
}`),
	result: &types.InfoResponse{
		Archives: nil,
		Cache: types.Cache{
			Path: "/home/test/.cache/borg/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
			Stats: types.Stats{
				TotalChunks:       0,
				TotalCSize:        0,
				TotalSize:         0,
				TotalUniqueChunks: 0,
				UniqueCSize:       0,
				UniqueSize:        0,
			},
		},
		Encryption: types.Encryption{
			Mode: "repokey-blake2",
		},
		Repository: types.Repository{
			ID:           "01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
			LastModified: "2024-12-02T10:28:45.000000",
			Location:     "/home/test/arcotest/dest/encrypted_with_123",
		},
		SecurityDir: "/home/test/.config/borg/security/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
	},
	wantErr: false,
}
var nonEmptyRepo = testBorgInfo{
	name: "Call info - non-empty repo",
	cmdResult: []byte(`
{
    "cache": {
        "path": "/home/test/.cache/borg/90cec45b910252da33ab8a2c90019ecfc8ad12ec59fd18d6e41f315d59775a9a",
        "stats": {
            "total_chunks": 8700014,
            "total_csize": 756393160032,
            "total_size": 947248071059,
            "total_unique_chunks": 355416,
            "unique_csize": 288204487393,
            "unique_size": 298658888098
        }
    },
    "encryption": {
        "mode": "none"
    },
    "repository": {
        "id": "90cec45b910252da33ab8a2c90019ecfc8ad12ec59fd18d6e41f315d59775a9a",
        "last_modified": "2024-12-02T12:14:45.000000",
        "location": "/home/test/arcotest/dest/main"
    },
    "security_dir": "/home/test/.config/borg/security/90cec45b910252da33ab8a2c90019ecfc8ad12ec59fd18d6e41f315d59775a9a"
}`),
	result: &types.InfoResponse{
		Archives: nil,
		Cache: types.Cache{
			Path: "/home/test/.cache/borg/90cec45b910252da33ab8a2c90019ecfc8ad12ec59fd18d6e41f315d59775a9a",
			Stats: types.Stats{
				TotalChunks:       8700014,
				TotalCSize:        756393160032,
				TotalSize:         947248071059,
				TotalUniqueChunks: 355416,
				UniqueCSize:       288204487393,
				UniqueSize:        298658888098,
			},
		},
		Encryption: types.Encryption{
			Mode: "none",
		},
		Repository: types.Repository{
			ID:           "90cec45b910252da33ab8a2c90019ecfc8ad12ec59fd18d6e41f315d59775a9a",
			LastModified: "2024-12-02T12:14:45.000000",
			Location:     "/home/test/arcotest/dest/main",
		},
		SecurityDir: "/home/test/.config/borg/security/90cec45b910252da33ab8a2c90019ecfc8ad12ec59fd18d6e41f315d59775a9a",
	},
}
var withWarning = testBorgInfo{
	name: "Call info - with warning",
	cmdResult: []byte(`
Using a pure-python msgpack! This will result in lower performance.
{
    "cache": {
        "path": "/home/test/.cache/borg/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
        "stats": {
            "total_chunks": 0,
            "total_csize": 0,
            "total_size": 0,
            "total_unique_chunks": 0,
            "unique_csize": 0,
            "unique_size": 0
        }
    },
    "encryption": {
        "mode": "repokey-blake2"
    },
    "repository": {
        "id": "01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
        "last_modified": "2024-12-02T10:28:45.000000",
        "location": "/home/test/arcotest/dest/encrypted_with_123"
    },
    "security_dir": "/home/test/.config/borg/security/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf"
}`),
	result: &types.InfoResponse{
		Archives: nil,
		Cache: types.Cache{
			Path: "/home/test/.cache/borg/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
			Stats: types.Stats{
				TotalChunks:       0,
				TotalCSize:        0,
				TotalSize:         0,
				TotalUniqueChunks: 0,
				UniqueCSize:       0,
				UniqueSize:        0,
			},
		},
		Encryption: types.Encryption{
			Mode: "repokey-blake2",
		},
		Repository: types.Repository{
			ID:           "01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
			LastModified: "2024-12-02T10:28:45.000000",
			Location:     "/home/test/arcotest/dest/encrypted_with_123",
		},
		SecurityDir: "/home/test/.config/borg/security/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
	},
	wantErr: false,
}

var withMultilineWarning = testBorgInfo{
	name: "Call info - multiline warning",
	cmdResult: []byte(`
Using a pure-python msgpack! This will result in lower performance.
Using a pure-python msgpack! This will result in lower performance.
Using a pure-python msgpack! This will result in lower performance.
{
    "cache": {
        "path": "/home/test/.cache/borg/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
        "stats": {
            "total_chunks": 0,
            "total_csize": 0,
            "total_size": 0,
            "total_unique_chunks": 0,
            "unique_csize": 0,
            "unique_size": 0
        }
    },
    "encryption": {
        "mode": "repokey-blake2"
    },
    "repository": {
        "id": "01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
        "last_modified": "2024-12-02T10:28:45.000000",
        "location": "/home/test/arcotest/dest/encrypted_with_123"
    },
    "security_dir": "/home/test/.config/borg/security/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf"
}`),
	result: &types.InfoResponse{
		Archives: nil,
		Cache: types.Cache{
			Path: "/home/test/.cache/borg/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
			Stats: types.Stats{
				TotalChunks:       0,
				TotalCSize:        0,
				TotalSize:         0,
				TotalUniqueChunks: 0,
				UniqueCSize:       0,
				UniqueSize:        0,
			},
		},
		Encryption: types.Encryption{
			Mode: "repokey-blake2",
		},
		Repository: types.Repository{
			ID:           "01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
			LastModified: "2024-12-02T10:28:45.000000",
			Location:     "/home/test/arcotest/dest/encrypted_with_123",
		},
		SecurityDir: "/home/test/.config/borg/security/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
	},
	wantErr: false,
}

var withMultilineWarningAndSpaces = testBorgInfo{
	name: "Call info - multiline warning with spaces",
	cmdResult: []byte(`
Using a pure-python msgpack! This will result in lower performance.
Using a pure-python msgpack! This will result in lower performance.
Using a pure-python msgpack! This will result in lower performance.
  {
    "cache": {
        "path": "/home/test/.cache/borg/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
        "stats": {
            "total_chunks": 0,
            "total_csize": 0,
            "total_size": 0,
            "total_unique_chunks": 0,
            "unique_csize": 0,
            "unique_size": 0
        }
    },
    "encryption": {
        "mode": "repokey-blake2"
    },
    "repository": {
        "id": "01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
        "last_modified": "2024-12-02T10:28:45.000000",
        "location": "/home/test/arcotest/dest/encrypted_with_123"
    },
    "security_dir": "/home/test/.config/borg/security/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf"
}`),
	result: &types.InfoResponse{
		Archives: nil,
		Cache: types.Cache{
			Path: "/home/test/.cache/borg/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
			Stats: types.Stats{
				TotalChunks:       0,
				TotalCSize:        0,
				TotalSize:         0,
				TotalUniqueChunks: 0,
				UniqueCSize:       0,
				UniqueSize:        0,
			},
		},
		Encryption: types.Encryption{
			Mode: "repokey-blake2",
		},
		Repository: types.Repository{
			ID:           "01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
			LastModified: "2024-12-02T10:28:45.000000",
			Location:     "/home/test/arcotest/dest/encrypted_with_123",
		},
		SecurityDir: "/home/test/.config/borg/security/01267068cadb779b15931dc6ff82c1f4ccebc39acc1f1f51dc12d4bba3a0decf",
	},
	wantErr: false,
}

var withJsonError = testBorgInfo{
	name:      "Call info - with json error",
	cmdResult: []byte(`{{}`),
	result:    nil,
	wantErr:   true,
}

func TestBorgInfo(t *testing.T) {
	var b Borg
	var cr *mocks.MockCommandRunner

	var setup = func(t *testing.T, output []byte, err error) {
		logConfig := zap.NewDevelopmentConfig()
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		log, logErr := logConfig.Build()
		assert.NoError(t, logErr, "Failed to create logger")

		cr = mocks.NewMockCommandRunner(gomock.NewController(t))
		cr.EXPECT().Info(gomock.Any()).Return(output, err)
		b = NewBorg("borg", log.Sugar(), []string{}, cr)
	}

	tests := []testBorgInfo{
		emptyRepo,
		nonEmptyRepo,
		withWarning,
		withMultilineWarning,
		withMultilineWarningAndSpaces,
		withJsonError,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			setup(t, tt.cmdResult, nil)

			// ACT
			result, status := b.Info(context.Background(), "test-repo", "test-password", false)

			// ASSERT
			if tt.wantErr {
				assert.True(t, status.HasError(), "Expected error, got nil")
			} else {
				assert.True(t, status.IsCompletedWithSuccess(), "Expected success, got error: %v", status.GetError())
				assert.Equal(t, tt.result, result, "Info() = %v, want %v", result, tt.result)
			}
		})
	}
}
