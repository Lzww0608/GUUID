package guuid

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := New()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkGenerator_New(b *testing.B) {
	gen := NewGenerator()
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := gen.New()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkUUID_String(b *testing.B) {
	uuid, _ := New()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = uuid.String()
	}
}

func BenchmarkParse(b *testing.B) {
	s := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(s)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParse_NoHyphens(b *testing.B) {
	s := "f47ac10b58cc4372a5670e02b2c3d479"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(s)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUUID_MarshalText(b *testing.B) {
	uuid, _ := New()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := uuid.MarshalText()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUUID_UnmarshalText(b *testing.B) {
	text := []byte("f47ac10b-58cc-4372-a567-0e02b2c3d479")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var uuid UUID
		err := uuid.UnmarshalText(text)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUUID_MarshalBinary(b *testing.B) {
	uuid, _ := New()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := uuid.MarshalBinary()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUUID_UnmarshalBinary(b *testing.B) {
	uuid, _ := New()
	data, _ := uuid.MarshalBinary()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var u UUID
		err := u.UnmarshalBinary(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUUID_EncodeToHex(b *testing.B) {
	uuid, _ := New()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = uuid.EncodeToHex()
	}
}

func BenchmarkDecodeFromHex(b *testing.B) {
	s := "f47ac10b58cc4372a5670e02b2c3d479"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := DecodeFromHex(s)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUUID_EncodeToBase64(b *testing.B) {
	uuid, _ := New()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = uuid.EncodeToBase64()
	}
}

func BenchmarkDecodeFromBase64(b *testing.B) {
	uuid, _ := New()
	s := uuid.EncodeToBase64()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := DecodeFromBase64(s)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUUID_Compare(b *testing.B) {
	uuid1, _ := New()
	uuid2, _ := New()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = uuid1.Compare(uuid2)
	}
}

func BenchmarkUUID_Timestamp(b *testing.B) {
	uuid, _ := New()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = uuid.Timestamp()
	}
}

func BenchmarkUUID_Time(b *testing.B) {
	uuid, _ := New()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = uuid.Time()
	}
}

// Benchmark concurrent generation
func BenchmarkGenerator_NewConcurrent(b *testing.B) {
	gen := NewGenerator()
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := gen.New()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Benchmark for batch generation
func BenchmarkGenerator_NewBatch(b *testing.B) {
	gen := NewGenerator()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			_, err := gen.New()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
