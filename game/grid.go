package game

type Grid struct {
	AutoScoring [4][12]bool
	Nodes [4][12]NodeState
	AutoLvL1Count [2]int
	TeliopLvL1Count [2]int
}

//go:generate stringer -type=NodeState
type NodeState int

const (
	Empty NodeState = iota
	Coral 	
	NodeStateCount
)	

type Row int

const (
	lvl1 Row = iota
	lvl2
	lvl3
	lvl4
	rowCount
)

var autoPoints = map[Row]int{
	lvl1: 3,
	lvl2: 4,
	lvl3: 6,
	lvl4: 7,
}
var teleopPoints = map[Row]int{
	lvl1: 2,
	lvl2: 3,
	lvl3: 4,
	lvl4: 5,
}

var validGridNodeStates = createValidGridStates()

// Returns a map of valid node states for each row and column in the grid.
func ValidGridNodeStates() map[Row]map[int]map[NodeState]string {
	return validGridNodeStates
}

func (grid *Grid) AutoGamePiecePoints() int {
	points := 0
	for row := lvl1; row < rowCount; row++ {
		for column := 0; column < 12; column++ {
			autoPieces, _ := grid.numScoredAutoTeleopGamePieces(row, column)
			if autoPieces > 0 {
				points += autoPoints[row]
			}
		}
	}
	points += grid.AutoLvL1Count[0]*autoPoints[1]
	points += grid.AutoLvL1Count[1]*autoPoints[1]
	return points
}

func (grid *Grid) TeleopGamePiecePoints() int {
	points := 0
	for row := lvl1; row < rowCount; row++ {
		for column := 0; column < 12; column++ {
			autoPieces, teleopPieces := grid.numScoredAutoTeleopGamePieces(row, column)
			if autoPieces == 0 && teleopPieces > 0 {
				points += teleopPoints[row]
			}
		}
	}
	points += grid.TeliopLvL1Count[0]*teleopPoints[1]
	points += grid.TeliopLvL1Count[1]*teleopPoints[1]
	return points
}

/* func (grid *Grid) SuperchargedPoints() int {
	return 3 * grid.NumSuperchargedNodes()
}
 */
/* func (grid *Grid) NumSuperchargedNodes() int {
	if !grid.IsFull() {
		return 0
	}

	numSuperchargedNodes := 0
	for row := lvl1; row < rowCount; row++ {
		for column := 0; column < 12; column++ {
			if grid.numScoredGamePieces(row, column) > 1 {
				numSuperchargedNodes++
			}
		}
	}
	return numSuperchargedNodes
} */

/* func (grid *Grid) LinkPoints() int {
	return 5 * len(grid.Links())
} */

/* func (grid *Grid) Links() []Link {
	var links []Link
	for row := lvl1; row < rowCount; row++ {
		startColumn := 0
		for startColumn < 7 {
			isValidLink := true
			for column := startColumn; column < startColumn+3; column++ {
				if grid.numScoredGamePieces(row, column) == 0 {
					isValidLink = false
					break
				}
			}

			if isValidLink {
				link := Link{Row: row, StartColumn: startColumn}
				links = append(links, link)
				startColumn += 3
			} else {
				startColumn++
			}
		}
	}
	return links
} */

// Returns true if this grid contains enough scored nodes to activate the coopertition bonus (both alliances' grids must
// meet this condition for the bonus to be awarded).
/* func (grid *Grid) IsCoopertitionThresholdAchieved() bool {
	pieces := 0
	for row := lvl1; row < rowCount; row++ {
		for column := 3; column < 6; column++ {
			pieces += grid.numScoredGamePieces(row, column)
		}
	}

	return pieces >= 3
} */

func (grid *Grid) IsFull() bool {
	for row := lvl1; row < rowCount; row++ {
		for column := 0; column < 12; column++ {
			if grid.numScoredGamePieces(row, column) == 0 {
				return false
			}
		}
	}
	return true
}

// Returns the separate counts of scored auto and teleop game pieces in the given node, limiting them to valid values.
func (grid *Grid) numScoredAutoTeleopGamePieces(row Row, column int) (int, int) {
	if row < lvl1 || row > lvl4 || column < 0 || column > 11 {
		// This is not a valid node.
		return 0, 0
	}

	autoScoring := grid.AutoScoring[row][column]
	nodeState := grid.Nodes[row][column]
	if _, ok := ValidGridNodeStates()[row][column][nodeState]; nodeState <= Empty || !ok {
		// This is an empty or invalid node state.
		return 0, 0
	}

	var totalPieces int
	if nodeState == Coral {
		totalPieces = 1
	} /* else {
		totalPieces = 2
	} */
	autoPieces := 0
	if autoScoring {
		autoPieces = 1
	}

	return autoPieces, totalPieces - autoPieces
}

// Returns the total number of game pieces in the given node, limiting it to valid values.
func (grid *Grid) numScoredGamePieces(row Row, column int) int {
	autoPieces, teleopPieces := grid.numScoredAutoTeleopGamePieces(row, column)
	return autoPieces + teleopPieces
}

// Builds a map of valid node states for each row and column in the grid.
func createValidGridStates() map[Row]map[int]map[NodeState]string {
	validGridNodeStates := make(map[Row]map[int]map[NodeState]string)
	for row := lvl1; row < rowCount; row++ {
		validGridNodeStates[row] = make(map[int]map[NodeState]string)
		for column := 0; column < 12; column++ {
			validGridNodeStates[row][column] = make(map[NodeState]string)
			for nodeState := Empty; nodeState < NodeStateCount; nodeState++ {
				if nodeState != Empty && row != lvl1 {
					if nodeState != Coral{
						continue
					}
				}
				validGridNodeStates[row][column][nodeState] = nodeState.String()
			}
		}
	}
	return validGridNodeStates
}
