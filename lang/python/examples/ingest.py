# Data ingestion using the `ingest` module.
from syredb import ingest
import pandas as pd

# Get ingestion sources
sources = ingest.get_sources()

# Access data source
data_source = sources["data"]

# Process sources
df = pd.read_csv(data_source.source.path)

# Create data
data = ingest.Data()
data.set_values(df)
data.save()
