import type { Client } from '@urql/svelte';
import { unwrap } from './api';
import { CREATE_SESSION_MUTATION } from './queries';
import { rememberSession } from './sessions';
import type { Session } from './types';

/** localStorage key of the player name typed on the home page. */
export const NAME_KEY = 'drbrain-mazes.displayName';

/** The player name this browser last used ('' when none). */
export function savedName(): string {
	try {
		return typeof localStorage === 'undefined' ? '' : (localStorage.getItem(NAME_KEY) ?? '');
	} catch {
		return '';
	}
}

/**
 * Creates a session on a map for the given player name, remembers the name and the session id
 * (for the home "Continue" card) and returns the new session id.
 */
export async function startSession(client: Client, mapID: string, displayName: string): Promise<string> {
	const name = displayName.trim();
	try {
		localStorage.setItem(NAME_KEY, name);
	} catch {
		// Storage blocked: the name is only a convenience.
	}
	const data = await unwrap<{ createSession: Session }>(client.mutation(CREATE_SESSION_MUTATION, { mapID, displayName: name || null }));
	rememberSession(data.createSession.id);
	return data.createSession.id;
}
