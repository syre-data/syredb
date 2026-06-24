# Data ingestion using the `ingest` module.
from syredb import ingest
import pandas as pd

# Get ingestion sources
sources = ingest.get_sources()

# Process sources
df = pd.read_csv(sources["data"])

# Create data
data = ingest.Data()
data.set_values(df)
data.save()
