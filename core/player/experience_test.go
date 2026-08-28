package player

import "testing"

func TestExperienceLevelBoundaries(t *testing.T) {
	tests := []struct {
		level int32
		total int32
	}{
		{0, 0}, {1, 7}, {16, 352}, {17, 394}, {31, 1507}, {32, 1628},
	}
	for _, test := range tests {
		if got := ExperienceForLevel(test.level); got != test.total {
			t.Errorf("ExperienceForLevel(%d) = %d, want %d", test.level, got, test.total)
		}
		p := New([16]byte{}, "test", ClientEditionJava)
		p.SetTotalExperience(test.total)
		level, total, progress := p.ExperienceSnapshot()
		if level != test.level || total != test.total || progress != 0 {
			t.Errorf("level %d snapshot = %d, %d, %f", test.level, level, total, progress)
		}
	}
}

func TestExperienceProgressAndClamping(t *testing.T) {
	p := New([16]byte{}, "test", ClientEditionJava)
	p.SetTotalExperience(12)
	level, total, progress := p.ExperienceSnapshot()
	if level != 1 || total != 12 || progress != 0.5555556 {
		t.Fatalf("snapshot = %d, %d, %f", level, total, progress)
	}
	p.AddExperience(-100)
	if _, total, _ = p.ExperienceSnapshot(); total != 0 {
		t.Fatalf("negative clamp total = %d", total)
	}
	p.SetExperienceLevel(32)
	if level, total, progress = p.ExperienceSnapshot(); level != 32 || total != 1628 || progress != 0 {
		t.Fatalf("level set snapshot = %d, %d, %f", level, total, progress)
	}
}
