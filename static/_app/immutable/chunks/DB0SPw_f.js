var e={NOT_FOUND:`Not found`,SESSION_TERMINAL:`Game is over — reset to try again`,INVALID_ARGUMENT:`Invalid input`,FORBIDDEN:`Forbidden — check the admin key`,LIMIT_REACHED:`Limit reached`};function t(t){let n=t.find(e=>e&&(e.message||e.extensions?.code));if(!n)return null;let r=typeof n.extensions?.code==`string`?n.extensions.code:null,i=n.message??``;return{code:r,title:r&&e[r]||`Request failed`,message:i}}function n(e){if(Array.isArray(e)){let n=t(e);if(n)return n}else if(e&&typeof e==`object`){let n=e;if(n.graphQLErrors?.length){let e=t(n.graphQLErrors);if(e)return e}if(n.networkError)return{code:null,title:`Cannot reach the server`,message:n.networkError.message??``};if(n.message)return{code:null,title:`Request failed`,message:n.message}}return{code:null,title:`Request failed`,message:typeof e==`string`?e:``}}function r(e){switch(e){case`MOVE_LIMIT`:return`You ran out of moves before reaching the finish.`;default:return`The game ended without reaching the finish.`}}var i=`level x y`,a=`
  move
  direction
  from { ${i} }
  stepped { ${i} }
  to { ${i} }
  blocked
  via
  status
  lossReason
`,o=`glyph a { ${i} } b { ${i} }`,s=`
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
`,c=`
  id
  name
  description
  difficulty
  width
  height
  levelCount
  levels
  start { ${i} }
  finish { ${i} }
  portals { ${o} }
  maxMoves
  parMoves
  minLevelChanges
  forcedDescents
`,l=`
  id
  displayName
  mapId
  map { ${s} }
  view { level levelCount rows portals { ${o} } }
  state {
    pos { ${i} }
    moves
    status
    lossReason
    path { ${i} }
  }
  seq
  createdAt
  lastActionAt
  attempts
  bestMoves
  lastEvent { ${a} }
`,u=`
  query Maps {
    maps { ${s} }
  }
`;`${s}`;var d=`
  query AdminMapById($id: ID!) {
    adminMap(id: $id) { ${c} }
  }
`,f=`
  query Session($id: ID!) {
    session(id: $id) { ${l} }
  }
`,p=`
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
`,m=`
  mutation CreateSession($mapID: ID!, $displayName: String) {
    createSession(mapID: $mapID, displayName: $displayName) { ${l} }
  }
`;`${l}`;var h=`
  mutation Move($sessionID: ID!, $direction: Direction!) {
    move(sessionID: $sessionID, direction: $direction) { ${l} }
  }
`;`${l}`;var g=`
  mutation Reset($sessionID: ID!) {
    reset(sessionID: $sessionID) { ${l} }
  }
`,_=`
  subscription SessionUpdated($sessionID: ID!) {
    sessionUpdated(sessionID: $sessionID) {
      seq
      session { ${l} }
      event { ${a} }
    }
  }
`,v=`
  mutation CreateMap($map: MapInput!) {
    createMap(map: $map) { ${c} }
  }
`,y=`
  mutation UpdateMap($map: MapInput!) {
    updateMap(map: $map) { ${c} }
  }
`,b=`
  mutation DeleteMap($id: ID!) {
    deleteMap(id: $id)
  }
`,x=`
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
`;export{u as a,p as c,y as d,x as f,b as i,f as l,r as m,v as n,h as o,n as p,m as r,g as s,d as t,_ as u};