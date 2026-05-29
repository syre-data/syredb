# Inserting data using the `api` module.
# %%
import datetime as dt
from syredb import api

# %%
expiration = dt.datetime.now(tz=dt.timezone.utc) + dt.timedelta(hours=1)
c = api.Client("127.0.0.1:3000", "owner@t.com", "root", expiration)

# %%
data = api.Data(c, "PL", "__web_client__")
# %%
data.set_values(
    {"wavelength": [11, 12, 13, 14, 15], "counts": [100, 110, 150, 110, 100]}
)
# %%
c.insert(data)

# %%
