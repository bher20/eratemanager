import logging
import aiohttp
from datetime import timedelta
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator, UpdateFailed
from .const import DEFAULT_SCAN_INTERVAL

_LOGGER = logging.getLogger(__name__)

class EnergyRatesCoordinator(DataUpdateCoordinator):
    def __init__(self, hass, entry):
        self.url = entry.data["url"]
        super().__init__(
            hass,
            _LOGGER,
            name="HA Energy Rates",
            update_interval=timedelta(seconds=DEFAULT_SCAN_INTERVAL),
        )

    async def _async_update_data(self):
        try:
            async with aiohttp.ClientSession() as session:
                async with session.get(self.url, timeout=10) as resp:
                    if resp.status != 200:
                        raise UpdateFailed(f"HTTP {resp.status}")
                    return await resp.json()
        except Exception as err:
            raise UpdateFailed(err)
