const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/**
 * Short relative time for activity columns: "just now", "2m ago", "3h ago", "5d ago" (under a
 * week), then a date ("Oct 4", or "Oct 4, 2025" in another year). Unparseable input is returned as is.
 */
export function relativeTime(iso: string, now: Date = new Date()): string {
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return iso;
	const sec = Math.round((now.getTime() - d.getTime()) / 1000);
	if (sec < 45) return 'just now';
	const min = Math.round(sec / 60);
	if (min < 60) return `${min}m ago`;
	const hours = Math.round(min / 60);
	if (hours < 24) return `${hours}h ago`;
	const days = Math.round(hours / 24);
	if (days < 7) return `${days}d ago`;
	const date = `${MONTHS[d.getMonth()]} ${d.getDate()}`;
	return d.getFullYear() === now.getFullYear() ? date : `${date}, ${d.getFullYear()}`;
}

/** Full local date and time for a `title` tooltip next to a relative time. */
export function absoluteTime(iso: string): string {
	const d = new Date(iso);
	return Number.isNaN(d.getTime()) ? iso : d.toLocaleString();
}
