package selection

import (
	"testing"
)

func TestPos(t *testing.T) {
	tests := []struct {
		name string
		pos  Pos
		want Pos
	}{
		{
			name: "zero position",
			pos:  Pos{Line: 0, Col: 0},
			want: Pos{Line: 0, Col: 0},
		},
		{
			name: "arbitrary position",
			pos:  Pos{Line: 5, Col: 10},
			want: Pos{Line: 5, Col: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pos != tt.want {
				t.Errorf("Pos = %+v, want %+v", tt.pos, tt.want)
			}
		})
	}
}

func TestRegion(t *testing.T) {
	tests := []struct {
		name   string
		region Region
		want   Region
	}{
		{
			name: "character-wise region",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 5},
				End:   Pos{Line: 0, Col: 10},
			},
			want: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 5},
				End:   Pos{Line: 0, Col: 10},
			},
		},
		{
			name: "line-wise region",
			region: Region{
				Kind:  KindLine,
				Start: Pos{Line: 2, Col: 0},
				End:   Pos{Line: 5, Col: 0},
			},
			want: Region{
				Kind:  KindLine,
				Start: Pos{Line: 2, Col: 0},
				End:   Pos{Line: 5, Col: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.region.Kind != tt.want.Kind {
				t.Errorf("Kind = %v, want %v", tt.region.Kind, tt.want.Kind)
			}
			if tt.region.Start != tt.want.Start {
				t.Errorf("Start = %+v, want %+v", tt.region.Start, tt.want.Start)
			}
			if tt.region.End != tt.want.End {
				t.Errorf("End = %+v, want %+v", tt.region.End, tt.want.End)
			}
		})
	}
}

func TestExtractRegion_CharWise_SingleLine(t *testing.T) {
	lines := []string{
		"hello world",
	}

	tests := []struct {
		name    string
		region  Region
		want    string
		wantErr bool
	}{
		{
			name: "extract substring",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 0},
				End:   Pos{Line: 0, Col: 5},
			},
			want:    "hello",
			wantErr: false,
		},
		{
			name: "extract single char",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 6},
				End:   Pos{Line: 0, Col: 7},
			},
			want:    "w",
			wantErr: false,
		},
		{
			name: "extract to end of line",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 6},
				End:   Pos{Line: 0, Col: 11},
			},
			want:    "world",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractRegion(lines, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractRegion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractRegion_CharWise_MultiLine(t *testing.T) {
	lines := []string{
		"first line",
		"second line",
		"third line",
	}

	tests := []struct {
		name    string
		region  Region
		want    string
		wantErr bool
	}{
		{
			name: "extract across two lines",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 6},
				End:   Pos{Line: 1, Col: 6},
			},
			want:    "line\nsecond",
			wantErr: false,
		},
		{
			name: "extract across three lines",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 0},
				End:   Pos{Line: 2, Col: 5},
			},
			want:    "first line\nsecond line\nthird",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractRegion(lines, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractRegion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractRegion_LineWise(t *testing.T) {
	lines := []string{
		"first line",
		"second line",
		"third line",
	}

	tests := []struct {
		name    string
		region  Region
		want    string
		wantErr bool
	}{
		{
			name: "extract single line",
			region: Region{
				Kind:  KindLine,
				Start: Pos{Line: 1, Col: 0},
				End:   Pos{Line: 1, Col: 0},
			},
			want:    "second line",
			wantErr: false,
		},
		{
			name: "extract multiple lines",
			region: Region{
				Kind:  KindLine,
				Start: Pos{Line: 0, Col: 0},
				End:   Pos{Line: 2, Col: 0},
			},
			want:    "first line\nsecond line\nthird line",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractRegion(lines, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractRegion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractRegion_Reversed(t *testing.T) {
	lines := []string{
		"hello world",
	}

	tests := []struct {
		name    string
		region  Region
		want    string
		wantErr bool
	}{
		{
			name: "reversed selection same line",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 11},
				End:   Pos{Line: 0, Col: 6},
			},
			want:    "world",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractRegion(lines, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractRegion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractRegion_UTF8(t *testing.T) {
	lines := []string{
		"hello 世界",
		"café",
	}

	tests := []struct {
		name    string
		region  Region
		want    string
		wantErr bool
	}{
		{
			name: "extract UTF-8 chars",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 6},
				End:   Pos{Line: 0, Col: 8},
			},
			want:    "世界",
			wantErr: false,
		},
		{
			name: "extract with UTF-8 combining chars",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 1, Col: 0},
				End:   Pos{Line: 1, Col: 4},
			},
			want:    "café",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractRegion(lines, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractRegion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractRegion_EdgeCases(t *testing.T) {
	lines := []string{
		"hello",
	}

	tests := []struct {
		name    string
		region  Region
		lines   []string
		want    string
		wantErr bool
	}{
		{
			name: "empty lines",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 0},
				End:   Pos{Line: 0, Col: 1},
			},
			lines:   []string{},
			want:    "",
			wantErr: true,
		},
		{
			name: "line out of bounds",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 10, Col: 0},
				End:   Pos{Line: 10, Col: 1},
			},
			lines:   lines,
			want:    "",
			wantErr: true,
		},
		{
			name: "col out of bounds",
			region: Region{
				Kind:  KindChar,
				Start: Pos{Line: 0, Col: 0},
				End:   Pos{Line: 0, Col: 100},
			},
			lines:   lines,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractRegion(tt.lines, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractRegion() = %q, want %q", got, tt.want)
			}
		})
	}
}
