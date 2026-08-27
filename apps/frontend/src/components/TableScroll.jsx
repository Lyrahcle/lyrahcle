export default function TableScroll({ children }) {
  return (
    <div className="-mx-1 overflow-x-auto px-1">
      <div className="min-w-[560px]">{children}</div>
    </div>
  )
}
