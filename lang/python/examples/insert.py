# Inserting data using the `api` module.
# %%
import datetime as dt
from syredb import api

# %%
expiration = dt.datetime.now(tz=dt.timezone.utc) + dt.timedelta(hours=1)
c = api.Client("127.0.0.1:3000", "owner@t.com", "root", expiration)

# %%
data = api.Data(c, "PL", "pl_setup")
# %%
data.set_values(
    {"wavelength": [500, 600, 700, 800, 900], "counts": [1000, 1100, 1500, 1100, 1000]}
)
# %%
c.insert(data)

# %%
