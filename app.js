(function() {
	'use strict';

	function buildTree(nodes) {
		var nodeMap = {};
		for(var i in nodes) {
			var node = nodes[i].vertex;
			node.children = [];
			nodeMap[node._id] = node

			var edges = nodes[i].path.edges
			if(edges.length <= 0) {
				continue;
			}
			nodeMap[edges[edges.length-1]._from].children.push(node);
		}
		return nodes[0].vertex;
	}

	var nodeTypeImplementations = {
		"all": function(node, doc) {
			var res = {};
			var success = true;
			for(var i in node.children) {
				var subres = runTree(node.children[i], doc);
				for(var j in subres) {
					res[j] = subres[j];
				}
				success = success && subres[node.children[i]._id].success;
			}
			res[node._id] = {
				tags: node.tags,
				success: success
			};
			return res;
		},
		"any": function(node, doc) {
			var res = {};
			var success = false;
			for(var i in node.children) {
				var subres = runTree(node.children[i], doc);
				for(var j in subres) {
					res[j] = subres[j];
				}
				success = success || subres[node.children[i]._id].success;
			}
			res[node._id] = {
				tags: node.tags,
				success: success
			};
			return res;
		},
		"script": function(node, doc) {
			var res = {}
			res[node._id] = {
				tags: node.tags,
				success: eval(node.content)
			};
			return res;
		},
	};

	function runTree(tree, doc) {
		if(!nodeTypeImplementations.hasOwnProperty(tree.type)) {
			throw "Unknown node type";
		}

		return nodeTypeImplementations[tree.type](tree, doc)
	}

	var Foxx = require('org/arangodb/foxx'),
		controller = new Foxx.Controller(applicationContext),
		db = require('org/arangodb').db;

	controller.post('/:label', function(req, res) {
		var obj = db._createStatement({
			'query': 'FOR n IN nodes FOR t IN n.tags FILTER t == @name RETURN n',
			'bindVars': {
				'name': 'name:'+req.params('label')
			}
		}).execute().toArray();
		obj = obj[0];

		var nodes = db._createStatement({
			'query': 'FOR p IN TRAVERSAL(nodes, edges, @startid, "outbound", {paths: true}) RETURN p',
			'bindVars': {
				'startid': obj._id
			}
		}).execute().toArray();

		var tree = buildTree(nodes);
		var result = runTree(tree, req.body());

		res.set('Content-Type', 'text/plain');
		res.body = JSON.stringify(result)
	}); 

}());
