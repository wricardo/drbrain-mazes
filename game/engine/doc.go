// Package engine implements the Dr Brain - Mazes rules: map parsing and
// validation, the move rules (walk, stairs, portals) and a shortest-path
// solver. It has no dependencies, globals or I/O.
//
// Coordinates: x = column, y = row, row 0 is the top of a level and y grows
// downward. Level 0 is the bottom of the tower; stairs up go to level+1.
package engine
