# Health Spending Simulation

CLI application for running fast concurrent simulations.
Runs 1000 simulations based on discrete probability distribution,
company size and threshhold amount.

## Build

### Windows

```
make build_windows
```

## Configure

Probabilities and amount ranges can be updated by editing `range_probabilities.csv`

## Run

### Windows

```
sim.exe  
```

Enter number of members and threshold when prompted

Results of simulation will be output as a CSV file under `results` folder

