import { Check, ChevronDown } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";

import { cn } from "@/lib/utils";

export type SelectOption = {
  label: string;
  value: string;
};

export function Select({
  className,
  onValueChange,
  options,
  placeholder,
  value,
}: {
  className?: string;
  onValueChange: (value: string) => void;
  options: SelectOption[];
  placeholder: string;
  value: string;
}) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const listboxId = useId();
  const selected = options.find((option) => option.value === value);

  useEffect(() => {
    function closeOnOutsideClick(event: MouseEvent) {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    document.addEventListener("mousedown", closeOnOutsideClick);
    return () => document.removeEventListener("mousedown", closeOnOutsideClick);
  }, []);

  return (
    <div className={cn("relative min-w-0", className)} ref={rootRef}>
      <button
        aria-controls={listboxId}
        aria-expanded={open}
        className="flex h-10 w-full min-w-0 items-center justify-between gap-2 rounded-none border border-zinc-200 bg-white px-3 text-left text-sm outline-none transition hover:bg-zinc-50 focus:border-zinc-400 focus:ring-4 focus:ring-zinc-100"
        onClick={() => setOpen((current) => !current)}
        type="button"
      >
        <span className={cn("truncate", selected ? "text-zinc-950" : "text-zinc-500")}>
          {selected?.label ?? placeholder}
        </span>
        <ChevronDown className="shrink-0 text-zinc-500" size={15} />
      </button>
      {open ? (
        <div
          className="absolute top-[calc(100%+4px)] left-0 z-50 max-h-72 w-full min-w-0 overflow-auto rounded-none border border-zinc-200 bg-white py-1 shadow-lg"
          id={listboxId}
          role="listbox"
        >
          {options.map((option) => {
            const selectedOption = option.value === value;

            return (
              <button
                aria-selected={selectedOption}
                className={cn(
                  "flex min-h-9 w-full items-center gap-2 px-3 py-2 text-left text-sm outline-none transition hover:bg-zinc-50 focus:bg-zinc-50",
                  selectedOption ? "font-medium text-zinc-950" : "text-zinc-700",
                )}
                key={option.value || "__empty"}
                onClick={() => {
                  onValueChange(option.value);
                  setOpen(false);
                }}
                role="option"
                type="button"
              >
                <Check
                  className={cn("shrink-0", selectedOption ? "opacity-100" : "opacity-0")}
                  size={14}
                />
                <span className="min-w-0 truncate">{option.label}</span>
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
