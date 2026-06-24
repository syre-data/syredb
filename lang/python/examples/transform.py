# Data type transform using the `transform` module.
from syredb import transform

# Read tabular data
data = transform.get_data()

# Get data as a pandas dataframe
df = data.as_pandas()

# Perform a calculation using a property of the data
df_avg = df / data.properties["sample_size"]

# Create the output data
out = transform.OutputData()
out.set_values(df_avg)

# Save the new data to the database
out.save()
