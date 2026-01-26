package op

import (
	"bytes"
	"fmt"
	"testing"
)

func TestLookup(t *testing.T) {
	tests := []struct {
		name    string
		opcode  byte
		want    string
		wantErr bool
	}{
		{"const", byte(Const), "const", false},
		{"add", byte(Add), "add", false},
		{"undefined", 255, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := LookupDefinition(tt.opcode)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for opcode %d", tt.opcode)
				}
				return
			}
			if err != nil {
				t.Fatalf("lookup failed: %v", err)
			}
			if def.Name != tt.want {
				t.Fatalf("unexpected definition name: %s", def.Name)
			}
		})
	}
}

func TestMakeOpcodes(t *testing.T) {
	tests := []struct {
		name     string
		opcode   Opcode
		operands []int
		want     []byte
	}{
		{"const", Const, []int{65535}, []byte{byte(Const), 255, 255}},
		{"add", Add, nil, []byte{byte(Add)}},
		{"undefined", Opcode(255), nil, []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Make(tt.opcode, tt.operands...)
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("unexpected instruction for %s.\nwant=%v\n got=%v", tt.name, tt.want, got)
			}
		})
	}
}

func TestReadOperands(t *testing.T) {
	tests := []struct {
		name    string
		opcode  Opcode
		operand int
	}{
		{"const", Const, 65535},
		{"jump", Jump, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ins := Make(tt.opcode, tt.operand)
			def, _ := LookupDefinition(ins[0])
			operands, read := ReadOperands(def, ins[1:])
			if read != 2 {
				t.Fatalf("expected to read 2 bytes, got %d", read)
			}
			if len(operands) != 1 || operands[0] != tt.operand {
				t.Fatalf("unexpected operands %v", operands)
			}
		})
	}
}

func TestInstructionsString(t *testing.T) {
	tests := []struct {
		name string
		ins  Instructions
		want string
	}{
		{"const+add", append(append(Instructions{}, Make(Const, 2)...), Make(Add)...), "0000 const 2\n0003 add\n"},
		{"jump", Instructions(Make(Jump, 5)), "0000 jump 5\n"},
		{"unknown", append(append(Instructions{}, Make(Const, 1)...), 255), "0000 const 1\nERROR: opcode 255 undefined\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ins.String(); got != tt.want {
				t.Fatalf("unexpected string.\nwant=%q\n got=%q", tt.want, got)
			}
		})
	}
}

func TestFmtInstructionOperandMismatch(t *testing.T) {
	ins := Instructions{}
	def := &Definition{Name: "const", OperandWidths: []int{2}}
	tests := []struct {
		name     string
		operands []int
		want     string
	}{
		{"too many", []int{1, 2}, "ERROR: operand len 2 does not match defined 1\n"},
		{"too few", []int{}, "ERROR: operand len 0 does not match defined 1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ins.fmtInstruction(def, tt.operands); got != tt.want {
				t.Fatalf("unexpected fmtInstruction result.\nwant=%q\n got=%q", tt.want, got)
			}
		})
	}
}

func TestReadUint16(t *testing.T) {
	tests := []struct {
		b    []byte
		want uint16
	}{
		{[]byte{1, 2}, 0x0102},
		{[]byte{255, 254}, 0xfffe},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i+1), func(t *testing.T) {
			if v := ReadUint16(tt.b); v != tt.want {
				t.Fatalf("unexpected uint16: %d", v)
			}
		})
	}
}
