/**
 * Echos back the request's body. The beginning of dynamic behavior.
 *
 * $ curl -d "echo? echo? echo..." localhost:5000/echo
 * echo? echo? echo...
 *
 */
Acme.Proxy.Echo = function() {
	this.handle = function(request) {
		var response = new AP.HTTP.Response();
		response.body = request.body;
		return response;
	};
}
