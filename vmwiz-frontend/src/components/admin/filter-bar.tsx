import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

/** Set of buttons to filter a list by status */
export function FilterBar<T extends string>({
    options,
    filter,
    onFilterChange,
    counts,
}: {
    options: readonly { value: T; label: string }[];
    filter: T;
    onFilterChange: (f: T) => void;
    counts: Record<T, number>;
}) {
    return (
        <div className="flex min-w-0 gap-1.5 overflow-x-auto">
            {options.map(({ value, label }) => (
                <Button
                    key={value}
                    variant={filter === value ? "default" : "outline"}
                    size="xs"
                    onClick={() => onFilterChange(value)}
                >
                    {label}
                    <Badge
                        variant="secondary"
                        className="ml-1.5 px-1.5 text-[0.65rem]"
                    >
                        {counts[value]}
                    </Badge>
                </Button>
            ))}
        </div>
    );
}
