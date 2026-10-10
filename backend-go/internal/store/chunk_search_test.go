package store_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVnBigramQuery(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	for q, want := range map[string]string{
		"cảnh báo học vụ":   `'canh' <-> 'bao' | 'bao' <-> 'hoc' | 'hoc' <-> 'vu'`,
		"Cảnh  báo, HỌC vụ": `'canh' <-> 'bao' | 'bao' <-> 'hoc' | 'hoc' <-> 'vu'`,
		"canh bao":          `'canh' <-> 'bao'`,
		"học":               `'hoc'`,
		"":                  ``,
		"  ?!  ":            ``,
		"a' | !b & (c)":     `'a' <-> 'b' | 'b' <-> 'c'`, // ký tự cú pháp tsquery bị loại
	} {
		var got string
		require.NoError(t, conn.QueryRow(ctx, `select vn_bigram_query($1)::text`, q).Scan(&got), q)
		require.Equal(t, want, got, q)
	}
	// NFD (dấu kết hợp) cho cùng kết quả với NFC
	var nfc, nfd string
	require.NoError(t, conn.QueryRow(ctx, `select vn_bigram_query('học vụ')::text, vn_bigram_query(normalize('học vụ', NFD))::text`).Scan(&nfc, &nfd))
	require.Equal(t, nfc, nfd)
	// tối đa 40 âm tiết: 41 từ vẫn dựng được và không lỗi
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select length(vn_bigram_query($1)::text)`, strings.TrimSpace(strings.Repeat("ab ", 60))).Scan(&n))
	require.Positive(t, n)
}

// TestSchemaChunkTSV / TestKeywordNFCandNFD / TestKeywordNoAccentMatches / TestTSVIndexUsed — US-P8-01 AC18.
func TestSchemaChunkTSV(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn) // có sẵn course + documents
	for i, txt := range []string{"Điều kiện bị cảnh báo học vụ được quy định tại Điều 5", "Cổng 8080 dùng cho dịch vụ web", "Ho\u0323c vu\u0323 ở dạng NFD"} {
		_, err := conn.Exec(ctx, `insert into content_chunks (document_id, course_ids, ord, text) values ($1, array[$2]::uuid[], $3, $4)`, f.doc, f.c1, i, txt)
		require.NoError(t, err)
	}
	var generated string
	require.NoError(t, conn.QueryRow(ctx, `select is_generated from information_schema.columns where table_name='content_chunks' and column_name='tsv'`).Scan(&generated))
	require.Equal(t, "ALWAYS", generated)
	var typ string
	require.NoError(t, conn.QueryRow(ctx, `select format_type(a.atttypid, a.atttypmod) from pg_attribute a where a.attrelid='public.content_chunks'::regclass and a.attname='tsv'`).Scan(&typ))
	require.Equal(t, "tsvector", typ)

	match := func(q string) []int {
		rows, err := conn.Query(ctx, `select ord from content_chunks where tsv @@ vn_bigram_query($1) order by ord`, q)
		require.NoError(t, err)
		defer rows.Close()
		var ords []int
		for rows.Next() {
			var o int
			require.NoError(t, rows.Scan(&o))
			ords = append(ords, o)
		}
		require.NoError(t, rows.Err())
		return ords
	}
	require.Equal(t, []int{0, 2}, match("điều kiện bị cảnh báo học vụ được quy định ở đâu"), "câu tự nhiên (OR các cặp: đoạn 2 khớp cặp học-vụ)")
	require.Equal(t, []int{0}, match("canh bao"), "không dấu khớp có dấu")
	require.Equal(t, []int{0, 2}, match("học vụ"), "NFC và NFD cùng khớp")
	require.Equal(t, []int{1}, match("cổng 8080"))
	require.Empty(t, match("quantum"))
}

func TestTSVIndexUsed(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	_, err := conn.Exec(ctx, `insert into content_chunks (document_id, course_ids, ord, text)
		select $1, array[$2]::uuid[], g, 'đoạn số ' || g || ' nói về chủ đề ' || (g % 97) from generate_series(1, 20000) g`, f.doc, f.c1)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `insert into content_chunks (document_id, course_ids, ord, text) values ($1, array[$2]::uuid[], 0, 'cảnh báo học vụ hiếm gặp')`, f.doc, f.c1)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `analyze content_chunks`)
	require.NoError(t, err)
	// ở 20.000 đoạn dễ quét tuần tự vẫn rẻ hơn; tắt Seq Scan để chứng minh chỉ mục dùng được cho đúng biểu thức `tsv @@ vn_bigram_query(…)`.
	_, err = conn.Exec(ctx, `set enable_seqscan = off`)
	require.NoError(t, err)
	rows, err := conn.Query(ctx, `explain select id from content_chunks where tsv @@ vn_bigram_query('cảnh báo học vụ')`)
	require.NoError(t, err)
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var l string
		require.NoError(t, rows.Scan(&l))
		plan.WriteString(l + "\n")
	}
	require.Contains(t, plan.String(), "content_chunks_tsv_gin", plan.String())
}
