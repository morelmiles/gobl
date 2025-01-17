package gobl_test

import (
	"io/ioutil"
	"testing"

	"github.com/invopop/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/dsig"
	"github.com/invopop/gobl/note"
)

var testKey = dsig.NewES256Key()

func TestEnvelop(t *testing.T) {
	msg := &note.Message{Content: "This is test content."}
	e, err := gobl.Envelop(msg)
	require.NoError(t, err)
	if assert.NotNil(t, e) {
		assert.Equal(t, "c6a5148ce90f70c24ebfe6de1abed0d0aafde4323a9bcf47cc4a5d544af9ea19", e.Head.Digest.Value)
	}
}

func TestEnvelopeDocument(t *testing.T) {
	m := new(note.Message)
	m.Content = "This is test content."

	e := gobl.NewEnvelope()
	if assert.NotNil(t, e.Head) {
		assert.NotEmpty(t, e.Head.UUID, "empty header uuid")
	}
	assert.NotNil(t, e.Document)

	if err := e.Insert(m); err != nil {
		t.Errorf("failed to insert payload: %v", err)
		return
	}

	if assert.NotNil(t, e.Head.Digest) {
		assert.Equal(t, e.Head.Digest.Algorithm, dsig.DigestSHA256, "unexpected digest algorithm")
		assert.Equal(t, "c6a5148ce90f70c24ebfe6de1abed0d0aafde4323a9bcf47cc4a5d544af9ea19", e.Head.Digest.Value, "digest should be the same")
	}

	assert.Empty(t, e.Signatures)
	assert.NoError(t, e.Sign(testKey), "signing envelope")
	assert.NotEmpty(t, e.Signatures, "expected a signature")

	assert.NoError(t, e.Validate(), "did not expect validation error")

	nm, ok := e.Extract().(*note.Message)
	require.True(t, ok, "unrecognized content")
	assert.Equal(t, m.Content, nm.Content, "content mismatch")
}

func TestEnvelopeExtract(t *testing.T) {
	e := &gobl.Envelope{}
	obj := e.Extract()
	assert.Nil(t, obj)
}

func TestEnvelopeInsert(t *testing.T) {
	m := new(note.Message)
	m.Content = "This is test content."

	t.Run("missing head", func(t *testing.T) {
		e := new(gobl.Envelope)
		err := e.Insert(m)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing head")
	})

	t.Run("no document", func(t *testing.T) {
		e := gobl.NewEnvelope()
		err := e.Insert(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no-document")
	})

}

func TestEnvelopeComplete(t *testing.T) {
	e := new(gobl.Envelope)

	data, err := ioutil.ReadFile("./samples/es/invoice-es-es.env.yaml")
	require.NoError(t, err)
	err = yaml.Unmarshal(data, e)
	require.NoError(t, err)

	err = e.Calculate()
	require.NoError(t, err)

	inv, ok := e.Extract().(*bill.Invoice)
	require.True(t, ok)
	require.NoError(t, err)

	assert.Equal(t, "1210.00", inv.Totals.Payable.String())
}

func TestEnvelopeCompleteErrors(t *testing.T) {
	t.Run("missing document", func(t *testing.T) {
		e := new(gobl.Envelope)
		err := e.Calculate()
		assert.Error(t, err)
		assert.ErrorIs(t, err, gobl.ErrNoDocument)
	})
	t.Run("missing document payload", func(t *testing.T) {
		e := gobl.NewEnvelope()
		err := e.Calculate()
		assert.Error(t, err)
		assert.ErrorIs(t, err, gobl.ErrNoDocument)
	})
}

func TestEnvelopeValidate(t *testing.T) {
	key := dsig.NewES256Key()
	tests := []struct {
		name string
		env  func() *gobl.Envelope
		want string
	}{
		{
			name: "empty envelope",
			env: func() *gobl.Envelope {
				return &gobl.Envelope{}
			},
			want: "$schema: cannot be blank; doc: cannot be blank; head: cannot be blank.",
		},
		{
			name: "missing message body, draft",
			env: func() *gobl.Envelope {
				env := gobl.NewEnvelope()
				env.Head.Draft = true
				require.NoError(t, env.Insert(&note.Message{}))
				return env
			},
			want: "doc: (content: cannot be blank.).",
		},
		{
			name: "missing sig, draft",
			env: func() *gobl.Envelope {
				env := gobl.NewEnvelope()
				env.Head.Draft = true
				require.NoError(t, env.Insert(&note.Message{Content: "foo"}))
				return env
			},
		},
		{
			name: "with sig, not draft",
			env: func() *gobl.Envelope {
				env := gobl.NewEnvelope()
				require.NoError(t, env.Insert(&note.Message{Content: "foo"}))
				assert.NoError(t, env.Sign(key))
				return env
			},
		},
		{
			name: "with sig, not draft, modified",
			env: func() *gobl.Envelope {
				env := gobl.NewEnvelope()
				require.NoError(t, env.Insert(&note.Message{Content: "foo"}))
				assert.NoError(t, env.Sign(key))
				msg := env.Extract().(*note.Message)
				msg.Content = "bar"
				return env
			},
			want: "document: digest mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.env()
			err := env.Validate()
			if tt.want == "" && err == nil {
				return
			}
			assert.EqualError(t, err, tt.want)
		})
	}
}

func TestEnvelopeSign(t *testing.T) {
	t.Run("cannot sign draft", func(t *testing.T) {
		env := gobl.NewEnvelope()
		env.Head.Draft = true
		require.NoError(t, env.Insert(&note.Message{}))
		err := env.Sign(testKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "draft: cannot sign draft envelope")
	})
	t.Run("cannot sign invalid document", func(t *testing.T) {
		env := gobl.NewEnvelope()
		require.NoError(t, env.Insert(&note.Message{})) // missing msg content
		err := env.Sign(testKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation: doc: (content: cannot be blank.).")
	})
	t.Run("sign valid document", func(t *testing.T) {
		env := gobl.NewEnvelope()
		require.NoError(t, env.Insert(&note.Message{Content: "Test Message"}))
		err := env.Sign(testKey)
		assert.NoError(t, err)
		assert.Len(t, env.Signatures, 1)
	})
}
