package crypto

import "testing"

func TestCRC32Base62_Deterministic(t *testing.T) {
	t.Parallel()
	a := CRC32Base62("hello_world", 6)
	b := CRC32Base62("hello_world", 6)
	if a != b {
		t.Errorf("CRC32Base62() not deterministic: %s != %s", a, b)
	}
}

func TestCRC32Base62_Width(t *testing.T) {
	t.Parallel()
	result := CRC32Base62("test", 6)
	if len(result) < 6 {
		t.Errorf("CRC32Base62() expected width >= 6, got %d", len(result))
	}
}

func TestCRC32Base62_DifferentInputs(t *testing.T) {
	t.Parallel()
	a := CRC32Base62("input_a", 6)
	b := CRC32Base62("input_b", 6)
	if a == b {
		t.Errorf("CRC32Base62() same output for different inputs: %s", a)
	}
}

func TestCRC32Base62_OnlyAlphanum(t *testing.T) {
	t.Parallel()
	result := CRC32Base62("some_data", 6)
	if !containsOnlyAlphanum(result) {
		t.Errorf("CRC32Base62() contains non-alphanumeric characters: %s", result)
	}
}
