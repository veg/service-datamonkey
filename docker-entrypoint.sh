#!/bin/sh
set -e

# This script runs as root initially to handle permissions

# Define a local writable path for the JWT key
LOCAL_JWT_KEY_PATH="/usr/local/etc/jwt/jwt_hs256.key"

# The JWT key should already be mounted at JWT_KEY_PATH via the volume mount
# Check if the key exists
if [ -f "$JWT_KEY_PATH" ]; then
    echo "JWT key found at $JWT_KEY_PATH"
    
    # Set permissions on the JWT key
    chmod 0600 "$JWT_KEY_PATH" 2>/dev/null || {
        echo "JWT key is read-only, copying to writable location: $LOCAL_JWT_KEY_PATH"
        cp "$JWT_KEY_PATH" "$LOCAL_JWT_KEY_PATH"
        chmod 0600 "$LOCAL_JWT_KEY_PATH"
        chown slurm:slurm "$LOCAL_JWT_KEY_PATH"
        
        # Update the environment variable to use the local copy
        export JWT_KEY_PATH="$LOCAL_JWT_KEY_PATH"
        echo "Using JWT key at: $JWT_KEY_PATH"
    }
else
    echo "Warning: JWT key not found at $JWT_KEY_PATH"
    # If no key is mounted, we could generate one here, but for now we'll just warn
fi

# Ensure data directories exist and have correct permissions
DATA_DIR=${DATASET_LOCATION:-/data/uploads}
JOB_DIR=${JOB_TRACKER_LOCATION:-/data/uploads}

# Create directories if they don't exist
mkdir -p "$DATA_DIR" "$JOB_DIR"

# Try to set permissions
chown -R slurm:slurm "$DATA_DIR" 2>/dev/null || echo "Warning: Could not change ownership on $DATA_DIR (likely mounted read-only)"
chmod -R 755 "$DATA_DIR" 2>/dev/null || echo "Warning: Could not change permissions on $DATA_DIR (likely mounted read-only)"

chown -R slurm:slurm "$JOB_DIR" 2>/dev/null || echo "Warning: Could not change ownership on $JOB_DIR (likely mounted read-only)"
chmod -R 755 "$JOB_DIR" 2>/dev/null || echo "Warning: Could not change permissions on $JOB_DIR (likely mounted read-only)"

# Check if we can write to the mounted volumes
if ! su -s /bin/sh slurm -c "touch \"$JOB_DIR/test_write_access\"" 2>/dev/null; then
    echo "Warning: Cannot write to $JOB_DIR as slurm user, using /tmp/jobs instead"
    export JOB_TRACKER_LOCATION="/tmp/jobs"
    echo "JOB_TRACKER_LOCATION set to $JOB_TRACKER_LOCATION"
else
    rm "$JOB_DIR/test_write_access"
    echo "Using $JOB_DIR for job tracking"
    
    # Create jobs.json only if using FileJobTracker
    if [ "${JOB_TRACKER_TYPE:-FileJobTracker}" = "FileJobTracker" ]; then
        if [ ! -f "$JOB_DIR/jobs.json" ]; then
            echo "{}" > "$JOB_DIR/jobs.json"
            chown slurm:slurm "$JOB_DIR/jobs.json" 2>/dev/null
            chmod 644 "$JOB_DIR/jobs.json" 2>/dev/null
            echo "Created jobs.json for FileJobTracker"
        fi
    fi
fi

if ! su -s /bin/sh slurm -c "touch \"$DATA_DIR/test_write_access\"" 2>/dev/null; then
    echo "Warning: Cannot write to $DATA_DIR as slurm user, using /tmp/datasets instead"
    export DATASET_LOCATION="/tmp/datasets"
    echo "DATASET_LOCATION set to $DATASET_LOCATION"
else
    rm "$DATA_DIR/test_write_access"
    echo "Using $DATA_DIR for dataset storage"
    
    # Create datasets.json only if using FileDatasetTracker
    if [ "${DATASET_TRACKER_TYPE:-FileDatasetTracker}" = "FileDatasetTracker" ]; then
        if [ ! -f "$DATA_DIR/datasets.json" ]; then
            echo "{}" > "$DATA_DIR/datasets.json"
            chown slurm:slurm "$DATA_DIR/datasets.json" 2>/dev/null
            chmod 644 "$DATA_DIR/datasets.json" 2>/dev/null
            echo "Created datasets.json for FileDatasetTracker"
        fi
    fi
fi

echo "Data directory setup complete"

# Switch to the slurm user and execute the main command
echo "Switching to slurm user..."
exec su -s /bin/sh slurm -c "$*"