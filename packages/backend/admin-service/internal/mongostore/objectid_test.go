package mongostore

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/unoadmin/admin-service/internal/domain"
)

func TestParseObjectIDRoundTrip(t *testing.T) {
	original := bson.NewObjectID()

	parsed, err := parseObjectID(original.Hex())
	if err != nil {
		t.Fatalf("parseObjectID() error = %v", err)
	}
	if parsed != original {
		t.Fatalf("parseObjectID() = %s, want %s", parsed.Hex(), original.Hex())
	}
	if got := objectIDHex(original); got != original.Hex() {
		t.Fatalf("objectIDHex() = %q, want %q", got, original.Hex())
	}
}

func TestParseObjectIDTreatsEmptyAsNil(t *testing.T) {
	parsed, err := parseObjectID("")
	if err != nil {
		t.Fatalf("parseObjectID(\"\") error = %v", err)
	}
	if !parsed.IsZero() {
		t.Fatalf("parseObjectID(\"\") = %s, want NilObjectID", parsed.Hex())
	}
	if hex := objectIDHex(bson.NilObjectID); hex != "" {
		t.Fatalf("objectIDHex(NilObjectID) = %q, want empty", hex)
	}
}

func TestParseObjectIDRejectsInvalidHex(t *testing.T) {
	if _, err := parseObjectID("not-an-object-id"); err == nil {
		t.Fatal("parseObjectID(invalid) = nil error, want ErrInvalid")
	} else if err != domain.ErrInvalid {
		t.Fatalf("parseObjectID(invalid) error = %v, want ErrInvalid", err)
	}
}

func TestParseObjectIDsSkipsEmptyValues(t *testing.T) {
	first := bson.NewObjectID()
	second := bson.NewObjectID()

	oids, err := parseObjectIDs([]string{first.Hex(), "", second.Hex()})
	if err != nil {
		t.Fatalf("parseObjectIDs() error = %v", err)
	}
	if len(oids) != 2 || oids[0] != first || oids[1] != second {
		t.Fatalf("parseObjectIDs() = %v, want [%s %s]", oids, first.Hex(), second.Hex())
	}
}

func TestConvertLegacyRefRewritesKnownReferences(t *testing.T) {
	migrated := bson.NewObjectID()
	mapping := map[string]bson.ObjectID{"tenant-holding": migrated}

	single := convertLegacyRef("tenant-holding", false, mapping)
	if oid, ok := single.(bson.ObjectID); !ok || oid != migrated {
		t.Fatalf("single ref = %v, want %s", single, migrated.Hex())
	}

	// 指向已删除记录的引用按空值处理，避免字符串混入 ObjectId 字段。
	unknown := convertLegacyRef("missing", false, mapping)
	if oid, ok := unknown.(bson.ObjectID); !ok || !oid.IsZero() {
		t.Fatalf("unknown ref = %v, want NilObjectID", unknown)
	}

	array := convertLegacyRef(bson.A{"tenant-holding", "missing"}, true, mapping)
	oids, ok := array.([]bson.ObjectID)
	if !ok || len(oids) != 1 || oids[0] != migrated {
		t.Fatalf("array ref = %v, want [%s]", array, migrated.Hex())
	}
}

func TestToStringSliceAcceptsDriverAndNativeSlices(t *testing.T) {
	cases := []any{
		bson.A{"a", "b"},
		[]any{"a", "b"},
		[]string{"a", "b"},
	}
	for _, input := range cases {
		if got := toStringSlice(input); len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("toStringSlice(%T) = %v, want [a b]", input, got)
		}
	}
}

func TestObjectIDHexesDropsNilValues(t *testing.T) {
	first := bson.NewObjectID()
	got := objectIDHexes([]bson.ObjectID{first, bson.NilObjectID})
	if len(got) != 1 || got[0] != first.Hex() {
		t.Fatalf("objectIDHexes() = %v, want [%s]", got, first.Hex())
	}
}
