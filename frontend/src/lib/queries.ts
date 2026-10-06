// GraphQL documents. Fields requested here must exist in graph/schema.graphqls.

const POSITION = `level x y`;

const MOVE_EVENT_FIELDS = `
  move
  direction
  from { ${POSITION} }
  stepped { ${POSITION} }
  to { ${POSITION} }
  blocked
  via
  status
  lossReason
`;

const PORTAL_FIELDS = `glyph a { ${POSITION} } b { ${POSITION} }`;

/** Public catalog fields: nothing beyond the start level. */
const MAP_FIELDS = `
  id
  name
  description
  difficulty
  width
  height
  levelCount
  preview
  maxMoves
  parMoves
  minLevelChanges
  forcedDescents
`;

/** Full layout, admin only (editor). */
const ADMIN_MAP_FIELDS = `
  id
  name
  description
  difficulty
  width
  height
  levelCount
  levels
  start { ${POSITION} }
  finish { ${POSITION} }
  portals { ${PORTAL_FIELDS} }
  maxMoves
  parMoves
  minLevelChanges
  forcedDescents
`;

/** Full session snapshot: everything the board, tower and counters need. */
export const SESSION_FIELDS = `
  id
  displayName
  mapId
  map { ${MAP_FIELDS} }
  view { level levelCount rows portals { ${PORTAL_FIELDS} } }
  state {
    pos { ${POSITION} }
    moves
    status
    lossReason
    path { ${POSITION} }
  }
  seq
  createdAt
  lastActionAt
  attempts
  bestMoves
  lastEvent { ${MOVE_EVENT_FIELDS} }
`;

export const MAPS_QUERY = `
  query Maps {
    maps { ${MAP_FIELDS} }
  }
`;

export const MAP_QUERY = `
  query MapById($id: ID!) {
    map(id: $id) { ${MAP_FIELDS} }
  }
`;

export const ADMIN_MAP_QUERY = `
  query AdminMapById($id: ID!) {
    adminMap(id: $id) { ${ADMIN_MAP_FIELDS} }
  }
`;

export const SESSION_QUERY = `
  query Session($id: ID!) {
    session(id: $id) { ${SESSION_FIELDS} }
  }
`;

export const SESSIONS_QUERY = `
  query Sessions($limit: Int, $sort: SessionSort, $mapId: ID) {
    sessions(sort: $sort, limit: $limit, mapId: $mapId) {
      id
      displayName
      mapId
      createdAt
      lastActionAt
      map { name parMoves }
      state { status moves }
    }
  }
`;

export const CREATE_SESSION_MUTATION = `
  mutation CreateSession($mapID: ID!, $displayName: String) {
    createSession(mapID: $mapID, displayName: $displayName) { ${SESSION_FIELDS} }
  }
`;

export const DELETE_SESSION_MUTATION = `
  mutation DeleteSession($id: ID!) {
    deleteSession(id: $id)
  }
`;

export const RENAME_SESSION_MUTATION = `
  mutation RenameSession($id: ID!, $displayName: String!) {
    renameSession(id: $id, displayName: $displayName) { ${SESSION_FIELDS} }
  }
`;

export const MOVE_MUTATION = `
  mutation Move($sessionID: ID!, $direction: Direction!) {
    move(sessionID: $sessionID, direction: $direction) { ${SESSION_FIELDS} }
  }
`;

export const BULK_MOVE_MUTATION = `
  mutation BulkMove($sessionID: ID!, $directions: [Direction!]!) {
    bulkMove(sessionID: $sessionID, directions: $directions) { ${SESSION_FIELDS} }
  }
`;

export const RESET_MUTATION = `
  mutation Reset($sessionID: ID!) {
    reset(sessionID: $sessionID) { ${SESSION_FIELDS} }
  }
`;

export const SESSION_UPDATED_SUBSCRIPTION = `
  subscription SessionUpdated($sessionID: ID!) {
    sessionUpdated(sessionID: $sessionID) {
      seq
      session { ${SESSION_FIELDS} }
      event { ${MOVE_EVENT_FIELDS} }
    }
  }
`;

export const CREATE_MAP_MUTATION = `
  mutation CreateMap($map: MapInput!) {
    createMap(map: $map) { ${ADMIN_MAP_FIELDS} }
  }
`;

export const UPDATE_MAP_MUTATION = `
  mutation UpdateMap($map: MapInput!) {
    updateMap(map: $map) { ${ADMIN_MAP_FIELDS} }
  }
`;

export const DELETE_MAP_MUTATION = `
  mutation DeleteMap($id: ID!) {
    deleteMap(id: $id)
  }
`;

export const VALIDATE_MAP_MUTATION = `
  mutation ValidateMap($map: MapInput!) {
    validateMap(map: $map) {
      valid
      issues { message }
      finishReachable
      parMoves
      minLevelChanges
      forcedDescents
    }
  }
`;
