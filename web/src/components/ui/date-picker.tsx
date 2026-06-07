import { CalendarDays, ChevronLeft, ChevronRight, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const dayLabels = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

function formatDateValue(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function parseDateValue(value: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return null;
  }
  const [year, month, day] = value.split("-").map(Number);
  const date = new Date(year, month - 1, day);

  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return null;
  }

  return date;
}

function monthLabel(date: Date) {
  return new Intl.DateTimeFormat("en-US", {
    month: "long",
    year: "numeric",
  }).format(date);
}

function visibleDays(monthDate: Date) {
  const firstDay = new Date(monthDate.getFullYear(), monthDate.getMonth(), 1);
  const start = new Date(firstDay);
  start.setDate(firstDay.getDate() - firstDay.getDay());

  return Array.from({ length: 42 }, (_, index) => {
    const day = new Date(start);
    day.setDate(start.getDate() + index);
    return day;
  });
}

export function DatePicker({
  className,
  max,
  min,
  onChange,
  placeholder,
  value,
}: {
  className?: string;
  max?: string;
  min?: string;
  onChange: (value: string) => void;
  placeholder: string;
  value: string;
}) {
  const selectedDate = parseDateValue(value);
  const minDate = min ? parseDateValue(min) : null;
  const maxDate = max ? parseDateValue(max) : null;
  const [open, setOpen] = useState(false);
  const [viewDate, setViewDate] = useState(selectedDate ?? new Date());
  const rootRef = useRef<HTMLDivElement>(null);
  const days = useMemo(() => visibleDays(viewDate), [viewDate]);
  const selectedTimestamp = selectedDate?.getTime();

  useEffect(() => {
    function closeOnOutsideClick(event: MouseEvent) {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    document.addEventListener("mousedown", closeOnOutsideClick);
    return () => document.removeEventListener("mousedown", closeOnOutsideClick);
  }, []);

  useEffect(() => {
    if (selectedTimestamp != null) {
      setViewDate(new Date(selectedTimestamp));
    }
  }, [selectedTimestamp]);

  function moveMonth(monthOffset: number) {
    setViewDate((current) => new Date(current.getFullYear(), current.getMonth() + monthOffset, 1));
  }

  return (
    <div className={cn("relative min-w-0", className)} ref={rootRef}>
      <button
        aria-expanded={open}
        className="flex h-10 w-full min-w-0 items-center justify-between gap-2 rounded-none border border-zinc-200 bg-white px-3 text-left text-sm outline-none transition hover:bg-zinc-50 focus:border-zinc-400 focus:ring-4 focus:ring-zinc-100"
        onClick={() => setOpen((current) => !current)}
        type="button"
      >
        <span className={cn("truncate", value ? "text-zinc-950" : "text-zinc-500")}>
          {value || placeholder}
        </span>
        <CalendarDays className="shrink-0 text-zinc-500" size={15} />
      </button>
      {open ? (
        <div className="absolute top-[calc(100%+4px)] left-0 z-50 w-[min(20rem,calc(100vw-2rem))] rounded-none border border-zinc-200 bg-white p-3 shadow-lg">
          <div className="mb-3 flex items-center justify-between gap-2">
            <Button
              aria-label="Previous month"
              onClick={() => moveMonth(-1)}
              size="icon-sm"
              type="button"
              variant="outline"
            >
              <ChevronLeft size={14} />
            </Button>
            <div className="font-medium text-sm">{monthLabel(viewDate)}</div>
            <Button
              aria-label="Next month"
              onClick={() => moveMonth(1)}
              size="icon-sm"
              type="button"
              variant="outline"
            >
              <ChevronRight size={14} />
            </Button>
          </div>
          <div className="grid grid-cols-7 gap-1 text-center text-[11px] text-zinc-500">
            {dayLabels.map((label) => (
              <div className="py-1" key={label}>
                {label}
              </div>
            ))}
          </div>
          <div className="mt-1 grid grid-cols-7 gap-1">
            {days.map((day) => {
              const dayValue = formatDateValue(day);
              const inCurrentMonth = day.getMonth() === viewDate.getMonth();
              const selected = dayValue === value;
              const disabled =
                (minDate ? day < minDate : false) || (maxDate ? day > maxDate : false);

              return (
                <button
                  className={cn(
                    "flex aspect-square items-center justify-center rounded-none text-sm outline-none transition focus:ring-2 focus:ring-zinc-300",
                    inCurrentMonth ? "text-zinc-800" : "text-zinc-300",
                    selected ? "bg-zinc-950 font-medium text-white" : "hover:bg-zinc-100",
                    disabled && "pointer-events-none opacity-35",
                  )}
                  disabled={disabled}
                  key={dayValue}
                  onClick={() => {
                    onChange(dayValue);
                    setOpen(false);
                  }}
                  type="button"
                >
                  {day.getDate()}
                </button>
              );
            })}
          </div>
          <div className="mt-3 flex items-center justify-between gap-2 border-zinc-200 border-t pt-3">
            <Button
              className="h-8"
              onClick={() => setViewDate(new Date())}
              type="button"
              variant="outline"
            >
              {"Today"}
            </Button>
            <Button
              className="h-8"
              disabled={!value}
              onClick={() => {
                onChange("");
                setOpen(false);
              }}
              type="button"
              variant="ghost"
            >
              <X size={14} />
              {"Clear"}
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
