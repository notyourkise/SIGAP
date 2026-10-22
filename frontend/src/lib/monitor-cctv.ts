// Helper pemilihan kamera untuk halaman Monitor CCTV.
//
// Dropdown kamera harus runtut berdasarkan kode (THD-02, THD-03, ...), dan
// kamera yang sudah dipakai pada sel lain tidak boleh muncul lagi agar satu
// kamera tidak tampil di dua sel sekaligus.

// Urutkan kode CCTV secara natural: THD-02 < THD-03 < THD-10 < THD-15.
export function sortCctvsByCode<T extends { kode_cctv: string }>(cctvs: readonly T[]): T[] {
  return [...cctvs].sort((a, b) =>
    a.kode_cctv.localeCompare(b.kode_cctv, 'id', {
      numeric: true,
      sensitivity: 'base',
    })
  );
}

// Daftar kamera yang boleh dipilih pada satu sel. Pilihan pada sel lain
// disembunyikan, tetapi pilihan sel itu sendiri tetap ditampilkan sebagai nilai
// aktif. Sel di luar layout yang sedang terlihat diabaikan.
export function getSelectableCctvs<T extends { id: string }>(
  cctvs: readonly T[],
  selection: Readonly<Record<number, string>>,
  cellIndex: number,
  visibleCells: readonly number[]
): T[] {
  const selectedElsewhere = new Set<string>();
  for (const index of visibleCells) {
    if (index === cellIndex) continue;
    const id = selection[index];
    if (id) selectedElsewhere.add(id);
  }
  return cctvs.filter((cctv) => !selectedElsewhere.has(cctv.id));
}

// Bersihkan selection lama: hanya simpan sel yang sedang terlihat dan buang
// duplikat. Bila terjadi duplikat, sel pertama yang dipertahankan.
export function uniqueSelectionForCells(
  selection: Readonly<Record<number, string>>,
  visibleCells: readonly number[]
): Record<number, string> {
  const next: Record<number, string> = {};
  const used = new Set<string>();

  for (const index of visibleCells) {
    const id = selection[index];
    if (!id || used.has(id)) continue;
    used.add(id);
    next[index] = id;
  }

  return next;
}
