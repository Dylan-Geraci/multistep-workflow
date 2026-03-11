interface Props {
  total: number;
  limit: number;
  offset: number;
  onChange: (offset: number) => void;
}

export default function Pagination({ total, limit, offset, onChange }: Props) {
  if (total <= limit) return null;

  const page = Math.floor(offset / limit) + 1;
  const totalPages = Math.ceil(total / limit);

  return (
    <div className="flex items-center justify-between pt-4">
      <span className="text-sm text-gray-500">
        Page {page} of {totalPages}
      </span>
      <div className="flex gap-2">
        <button
          disabled={offset === 0}
          onClick={() => onChange(Math.max(0, offset - limit))}
          className="rounded border border-gray-300 px-3 py-1 text-sm disabled:opacity-40"
        >
          Previous
        </button>
        <button
          disabled={offset + limit >= total}
          onClick={() => onChange(offset + limit)}
          className="rounded border border-gray-300 px-3 py-1 text-sm disabled:opacity-40"
        >
          Next
        </button>
      </div>
    </div>
  );
}
