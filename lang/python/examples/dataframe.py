import syredb

# Read tabular data
data = syredb.get_data()

# Get data as a pandas dataframe
df = data.as_pandas()

# Perform a calculation using a property of the data
df_avg = df / data.properties["sample_size"]

# Insert the new data back into the database
syredb.insert(df_avg)
