import voluptuous as vol
from homeassistant import config_entries
from .const import DOMAIN, PROVIDERS, DEFAULT_PROVIDER

class EnergyRatesConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    VERSION = 1

    async def async_step_user(self, user_input=None):
        if user_input is not None:
            self.provider = user_input["provider"]
            self.default_url = PROVIDERS[self.provider]["default_url"]
            return await self.async_step_url()

        schema = vol.Schema({
            vol.Required("provider", default=DEFAULT_PROVIDER):
                vol.In({k: v["name"] for k, v in PROVIDERS.items()})
        })
        return self.async_show_form(step_id="user", data_schema=schema)

    async def async_step_url(self, user_input=None):
        if user_input is not None:
            return self.async_create_entry(
                title=f"{PROVIDERS[self.provider]['name']} Energy Rates",
                data={
                    "provider": self.provider,
                    "url": user_input["url"],
                },
            )

        schema = vol.Schema({
            vol.Required("url", default=self.default_url): str
        })
        return self.async_show_form(step_id="url", data_schema=schema)
