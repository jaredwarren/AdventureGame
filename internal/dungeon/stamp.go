package dungeon

// StampOrthogonalMaze converts a cell maze into a dense tile grid of size
// (2*m.W+1)×(2*m.H+1). Odd-indexed “cells” are maze rooms; even indices are
// walls. grassGID / wallGID should match game tile GIDs (e.g. world.GIDGrass,
// world.GIDWall). Diagonal corners between two open passages are filled so
// movement doesn’t snag on single wall pixels.
func StampOrthogonalMaze(m *Maze, grassGID, wallGID int) (mapW, mapH int, data []int) {
	mapW = 2*m.W + 1
	mapH = 2*m.H + 1
	data = make([]int, mapW*mapH)
	for i := range data {
		data[i] = wallGID
	}
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			tx := 2*x + 1
			ty := 2*y + 1
			data[ty*mapW+tx] = grassGID
			if m.Pass[y][x][DirE] {
				data[ty*mapW+tx+1] = grassGID
			}
			if m.Pass[y][x][DirS] {
				data[(ty+1)*mapW+tx] = grassGID
			}
			if m.Pass[y][x][DirE] && m.Pass[y][x][DirS] {
				data[(ty+1)*mapW+tx+1] = grassGID
			}
			if m.Pass[y][x][DirE] && m.Pass[y][x][DirN] && ty >= 1 {
				data[(ty-1)*mapW+tx+1] = grassGID
			}
			if m.Pass[y][x][DirW] && m.Pass[y][x][DirS] && tx >= 1 {
				data[(ty+1)*mapW+tx-1] = grassGID
			}
			if m.Pass[y][x][DirW] && m.Pass[y][x][DirN] && tx >= 1 && ty >= 1 {
				data[(ty-1)*mapW+tx-1] = grassGID
			}
		}
	}
	return mapW, mapH, data
}
