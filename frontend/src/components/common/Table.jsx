export function Table({ headers, rows }) {
  return (
    <div className="table-card">
      <div className="table-card__head">
        {headers.map((header) => (
          <span key={header}>{header}</span>
        ))}
      </div>
      {rows}
    </div>
  );
}